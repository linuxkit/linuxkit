## How to build a CVM compatiable image

```sh
git clone https://github.com/linuxkit/linuxkit.git
cd linuxkit
git fetch origin pull/3947/head:cvm && git checkout -b cvm
make 

INITRD_LARGE_THAN_4GiB=1

if [ $INITRD_LARGE_THAN_4GiB -eq 1 ]; then
    (cd tools/grub && docker build -f Dockerfile.rhel -t linuxkit-hack/grub .)

    if [ -z $(docker ps -f name='registry' -q) ]; then
      docker run -d -p 5000:5000 --restart=always --name registry registry:2
    fi

    (
      remote_registry="localhost:5000/"
      tag="v0.1"
      cd tools/mkimage-raw-efi-ext4/ && 
      docker build . -t ${remote_registry}mkimage-raw-efi-ext4:$tag && 
      docker push ${remote_registry}mkimage-raw-efi-ext4:$tag
    )
    image_format="raw-efi-ext4"
else
    image_format="raw-efi"
fi

# build linux kernel
(
  cd contrib/foreign-kernels && 
  docker build -f Dockerfile.rpm.upstream-v6.3 . -t linuxkit/kernel:upstream-v6.3
)

# build a raw-efi image
bin/linuxkit build --docker examples/dm-crypt-cvm.yml -f $image_format

```
