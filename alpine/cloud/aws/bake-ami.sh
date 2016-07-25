#!/bin/sh

# Script to automate creation and snapshotting of a Moby AMI.  Currently, it's
# intended to be invoked from an instance running in the same region as the
# target AMI will be in, since it directly mounts the created EBS volume as a
# device on this running instance.

set -e

PROVIDER="aws"

. "./build-common.sh"
. "${MOBY_SRC_ROOT}/cloud/aws/common.sh"

# TODO(nathanleclaire): This could be calculated dynamically to avoid conflicts.
EBS_DEVICE=/dev/xvdb

bake_image() 
{
	# Create a new EBS volume.  We will format this volume to boot into Moby
	# initrd via syslinux in MBR.  That formatted drive can then be snapshotted
	# and turned into an AMI.
	VOLUME_ID=$(aws ec2 create-volume \
		--size 20 \
		--availability-zone $(current_instance_az) | jq -r .VolumeId)

	tag ${VOLUME_ID}

	aws ec2 wait volume-available --volume-ids ${VOLUME_ID}

	arrowecho "Attaching volume"
	aws ec2 attach-volume \
		--volume-id ${VOLUME_ID} \
		--device ${EBS_DEVICE} \
		--instance-id $(current_instance_id) >/dev/null

	aws ec2 wait volume-in-use --volume-ids ${VOLUME_ID}

	format_on_device "${EBS_DEVICE}"
	configure_syslinux_on_device_partition "${EBS_DEVICE}" "${EBS_DEVICE}1"

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

	echo "${IMAGE_ID}" >"${MOBY_SRC_ROOT}/cloud/aws/ami_id.out"
}

clean_tagged_resources()
{
	if [ -d "${MOBY_SRC_ROOT}/moby" ]
	then
		rm -rf "${MOBY_SRC_ROOT}/moby"
	fi

	VOLUME_ID=$(aws ec2 describe-volumes --filters "Name=tag-key,Values=$1" | jq -r .Volumes[0].VolumeId)
	if [ ${VOLUME_ID} = "null" ]
	then
		arrowecho "No volume found, skipping"
	else
		arrowecho "Detaching volume"
		aws ec2 detach-volume --volume-id ${VOLUME_ID} >/dev/null || errecho "WARN: Error detaching volume!"
		aws ec2 wait volume-available --volume-ids ${VOLUME_ID}
		arrowecho "Deleting volume"
		aws ec2 delete-volume --volume-id ${VOLUME_ID} >/dev/null
	fi

	IMAGE_ID=$(aws ec2 describe-images --filters "Name=tag-key,Values=$1" | jq -r .Images[0].ImageId)
	if [ ${IMAGE_ID} = "null" ]
	then
		arrowecho "No image found, skipping"
	else
		arrowecho "Deregistering previously baked AMI"

		# Sometimes describe-images does not return null even if the found
		# image cannot be deregistered 
		#
		# TODO(nathanleclaire): More elegant solution?
		aws ec2 deregister-image --image-id ${IMAGE_ID} >/dev/null || errecho "WARN: Issue deregistering previously tagged image!"
	fi

	SNAPSHOT_ID=$(aws ec2 describe-snapshots --filters "Name=tag-key,Values=$1" | jq -r .Snapshots[0].SnapshotId)
	if [ ${SNAPSHOT_ID} = "null" ]
	then
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
		arrowecho "Cleaning resources from previous build tag if applicable..."
		clean_tagged_resources "${TAG_KEY_PREV}"
		arrowecho "Cleaning resources from current build tag if applicable..."
		clean_tagged_resources "${TAG_KEY}"
		;;
	*)
		echo "Command $1 not found.  Usage: ./bake-ami.sh [bake|clean]"
esac
