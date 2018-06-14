# Using LinuxKit on Scaleway

This is a quick guide to run LinuxKit on Scaleway (only VPS x86_64 for now)

## Setup
Before you proceed it's recommanded that you set up the [Scaleway CLI](https://github.com/scaleway/scaleway-cli/)
and perform an `scw login`. This will create a `$HOME/.scwrc` file containing the required API token.

You can also use the `SCW_TOKEN` environment variable to set a Scaleway token. 
The `-token` flag of the `linuxkit push scaleway` and `linuxkit run scaleway` can also be used.

The environment variable `SCW_TARGET_REGION` is used to set the region (there is also the `-region` flag)


## Build an image

Scaleway requires a `iso-efi` image. To create one:

```
$ linuxkit build -format iso-efi examples/scaleway.yml
```

### Changes needed in the yaml

* You have to set `root=/dev/vda` in the `cmdline` to have the right device set on boot
* The metadata package is not only used to set the metadata, but also to signal Scaleway that the instance has booted. So it is encouraged to use it (dhcpcd must be set before)

## Push image

You have to do `linuxkit push scaleway scaleway.iso` to upload it to your Scaleway images.
By default the image name is the name of the ISO file without the extension. 
It can be overidden with the `-img-name` flag or the `SCW_IMAGE_NAME` environment variable.

**Note 1:** If an image (and snapshot) of the same name exists, it will be replaced.

**Note 2:** The image is region specific: if you create an image in `par1` you can't use is in `ams1`.

### Push process

Building a Scaleway image have a special process. Basically:

* Create an `image-builder` instance with an additional volume, based on Ubuntu Xenial (only x86_64 for now)
* Copy the ISO image on this instance
* Use `dd` to write the image on the additional volume (`/dev/vdb` by default)
* Terminate the instance, create a snapshot, and create an image from the snapshot

**Note 1:** An image is linked to a snapshot, so you can't delete a snapshot before the image.

**Note 2:** You can specify an already running instance to act as the image builder with the `-instance-id` flag. But if you don't specify the `-no-clean` flag it will be destroyed upon completion.

## Create an instance and connect to it

With the image created, we can now create an instance.

```
linuxkit run scaleway scaleway
```

By default, the instance name is `linuxkit`. It can be overidden with the `-instance-name` flag.
If you don't set the `-no-attach` flag, you will be connected to the serial port.

You can edit the Scaleway example to allow you to SSH to your instance in order to use it.
