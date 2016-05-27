#!/bin/bash

# Script to automate creation and snapshotting of a Moby AMI.  Currently, it's
# intended to be invoked from an instance running in the same region as the
# target AMI will be in, since it directly mounts the created EBS volume as a
# device on this running instance.

set -e

TAG_KEY=moby-bake
INSTANCE_ENDPOINT=http://169.254.169.254/latest
INSTANCE_METADATA_API_ENDPOINT=${INSTANCE_ENDPOINT}/meta-data/
IMAGE_NAME=${IMAGE_NAME:-"Moby Linux"}
IMAGE_DESCRIPTION=${IMAGE_DESCRIPTION:-"The best OS for running Docker"}

# TODO(nathanleclaire): This could be calculated dynamically to avoid conflicts.
EBS_DEVICE=/dev/xvdb

function arrowecho () {
    echo " --->" "$@"
}

function current_instance_region () {
    curl -s ${INSTANCE_ENDPOINT}/dynamic/instance-identity/document | jq .region -r
}

function current_instance_az () {
    curl -s ${INSTANCE_METADATA_API_ENDPOINT}/placement/availability-zone
}

function current_instance_id () {
    curl -s ${INSTANCE_METADATA_API_ENDPOINT}/instance-id
}

# We tag resources created as part of the build to ensure that they can be
# cleaned up later.
function tag () {
    arrowecho "Tagging $1"
    aws ec2 create-tags --resources "$1" --tags Key=${TAG_KEY},Value= >/dev/null
}

function format_device () {
    arrowecho "Waiting for EBS device to appear in build container"

    while [[ ! -e ${EBS_DEVICE} ]]; do
        sleep 1
    done

    # This heredoc might be confusing at first glance, so here is a detailed
    # summary of what each line does:
    #
    # n  - create new partition
    # p  - make it a primary partition
    # 1  - it should be partition #1
    # \n - use default first cylinder
    # \n - use default last cylinder
    # a  - toggle a partition as bootable
    # 1  - do the 1st partition specifically
    # w  - write changes and exit
    arrowecho "Formatting boot partition"
    fdisk ${EBS_DEVICE} >/dev/null << EOF
n
p
1


a
1
w
EOF

    # To ensure everything went smoothly, print the resulting partition table.
    echo
    arrowecho "Printing device partition contents"
    fdisk -l ${EBS_DEVICE}

    ROOT_PARTITION="${EBS_DEVICE}1"
    ROOT_PARTITION_MOUNT="/mnt/moby"

    # Mount created root partition, format it as ext4, and copy over the needed
    # files for boot (syslinux configuration, kernel binary, and initrd.img)
    arrowecho "Making filesystem on partition 1"
    mke2fs -t ext4 ${ROOT_PARTITION}

    arrowecho "Mounting partition filesystem"
    mkdir ${ROOT_PARTITION_MOUNT}
    mount -t ext4 ${ROOT_PARTITION} ${ROOT_PARTITION_MOUNT}

    arrowecho "Copying image and kernel binary to partition"

    # Get files needed to boot in place.
    cp /mnt/syslinux.cfg ${ROOT_PARTITION_MOUNT}
    cp /mnt/kernel/vmlinuz64 ${ROOT_PARTITION_MOUNT}
    cp /mnt/initrd.img ${ROOT_PARTITION_MOUNT}

    # From http://www.syslinux.org/wiki/index.php?title=EXTLINUX:
    #
    # "Note that EXTLINUX installs in the filesystem partition like a
    # well-behaved bootloader :). Thus, it needs a master boot record in the
    # partition table; the mbr.bin shipped with SYSLINUX should work well."

    # Thus, this step installs syslinux on the mounted filesystem (partition
    # 1).
    arrowecho "Installing syslinux to partition"
    extlinux --install ${ROOT_PARTITION_MOUNT} 

    # Format master boot record in partition table on target device.
    arrowecho "Copying MBR to partition table in target device"
    dd if=/usr/share/syslinux/mbr.bin of=${EBS_DEVICE} bs=440 count=1

    umount ${ROOT_PARTITION_MOUNT}

    arrowecho "Checking device/partition sanity"
    fdisk -l ${EBS_DEVICE}
}

function bake_image () {
    # Create a new EBS volume.  We will format this volume to boot into Moby
    # initrd via syslinux in MBR.  That formatted drive can then be snapshotted
    # and turned into an AMI.
    VOLUME_ID=$(aws ec2 create-volume \
        --size 1 \
        --availability-zone $(current_instance_az) | jq -r .VolumeId)

    tag ${VOLUME_ID}

    aws ec2 wait volume-available --volume-ids ${VOLUME_ID}

    arrowecho "Attaching volume"
    aws ec2 attach-volume \
        --volume-id ${VOLUME_ID} \
        --device ${EBS_DEVICE} \
        --instance-id $(current_instance_id) >/dev/null

    aws ec2 wait volume-in-use --volume-ids ${VOLUME_ID}

    format_device

    arrowecho "Taking snapshot!"

    # Take a snapshot of the volume we wrote to.
    SNAPSHOT_ID=$(aws ec2 create-snapshot \
        --volume-id ${VOLUME_ID} \
        --description "Snapshot of Moby device for AMI baking" | jq -r .SnapshotId)

    tag ${SNAPSHOT_ID}

    arrowecho "Waiting for snapshot completion"

    aws ec2 wait snapshot-completed --snapshot-ids ${SNAPSHOT_ID}

    # Convert that snapshot into an AMI as the root device.
    IMAGE_ID=$(aws ec2 register-image \
        --name "${IMAGE_NAME}" \
        --description "${IMAGE_DESCRIPTION}" \
        --architecture x86_64 \
        --root-device-name "${EBS_DEVICE}" \
        --virtualization-type "hvm" \
        --block-device-mappings "[
            {
                \"DeviceName\": \"${EBS_DEVICE}\",
                \"Ebs\": {
                    \"SnapshotId\": \"${SNAPSHOT_ID}\"
                }
            }
        ]" | jq -r .ImageId)

    tag ${IMAGE_ID}

    # Boom, now you (should) have a Moby AMI.
    arrowecho "Created AMI: ${IMAGE_ID}"

    echo ${IMAGE_ID} >/mnt/aws/ami_id.out
}

function clean_tagged_resources () {
    if [[ -d /mnt/moby ]]; then
        rm -rf /mnt/moby
    fi

    VOLUME_ID=$(aws ec2 describe-volumes --filters "Name=tag-key,Values=$TAG_KEY" | jq -r .Volumes[0].VolumeId)
    if [[ ${VOLUME_ID} == "null" ]]; then
        arrowecho "No volume found, skipping"
    else
        arrowecho "Detaching volume"
        aws ec2 detach-volume --volume-id ${VOLUME_ID} >/dev/null
        aws ec2 wait volume-available --volume-ids ${VOLUME_ID}
        arrowecho "Deleting volume"
        aws ec2 delete-volume --volume-id ${VOLUME_ID} >/dev/null
    fi

    IMAGE_ID=$(aws ec2 describe-images --filters "Name=tag-key,Values=$TAG_KEY" | jq -r .Images[0].ImageId)
    if [[ ${IMAGE_ID} == "null" ]]; then
        arrowecho "No image found, skipping"
    else
        arrowecho "Deregistering previously baked AMI"

        # Sometimes describe-images does not return null even if the found
        # image cannot be deregistered 
        #
        # TODO(nathanleclaire): More elegant solution?
        aws ec2 deregister-image --image-id ${IMAGE_ID} >/dev/null || true
    fi

    SNAPSHOT_ID=$(aws ec2 describe-snapshots --filters "Name=tag-key,Values=$TAG_KEY" | jq -r .Snapshots[0].SnapshotId)
    if [[ ${SNAPSHOT_ID} == "null" ]]; then
        arrowecho "No snapshot found, skipping"
    else
        arrowecho "Deleting volume snapshot"
        aws ec2 delete-snapshot --snapshot-id ${SNAPSHOT_ID}
    fi
}

case "$1" in
    bake)
        bake_image
        ;;
    clean)
        clean_tagged_resources
        ;;
    regions)
        echo '"AWSRegionArch2AMI": {'
        for REGION in us-west-1 us-west-2 us-east-1 eu-west-1 eu-central-1 ap-southeast-1 ap-northeast-1 ap-southeast-2 ap-northeast-2 sa-east-1; do
            REGION_AMI_ID=$(aws ec2 copy-image \
                --source-region $(current_instance_region) \
                --source-image-id $(cat /mnt/aws/ami_id.out) \
                --region "${REGION}" \
                --name "${IMAGE_NAME}" \
                --description "${IMAGE_DESCRIPTION}" | jq -r .ImageId)
            echo "    \"${REGION}\": {
        \"HVM64\": \"${REGION_AMI_ID}\",
        \"HVMG2\": \"NOT_SUPPORTED\"
    },"
        done
        echo "}"
        arrowecho "All done.  Make sure to remove the trailing comma."
        ;;
    *)
        arrowecho "Command $1 not found.  Usage: ./bake-ami.sh [bake|clean|regions]"
esac
