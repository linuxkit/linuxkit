# Using LinuxKit on Amazon Web Services (AWS)

This is a quick guide to run LinuxKit on AWS.

## Setup

Before you proceed it's recommended that you set up the [AWS CLI](https://aws.amazon.com/cli/)
and perform an `aws configure`

If you do not wish to install these tools you should ensure that you set the AWS [environment variables](http://docs.aws.amazon.com/cli/latest/userguide/cli-environment.html)

You will need to create an Amazon S3 Storage Bucket for your LinuxKit images and create a VM Import Service Role.
Instructions on how to do this can be found [here](http://docs.aws.amazon.com/vm-import/latest/userguide/vmimport-image-import.html#w2ab1c10c15b7).

Finally, you must set the `AWS_REGION` environment variable as this is used by the AWS Go SDK.
```
export AWS_REGION=eu-west-1
```

## Build an image

AWS requires a `RAW` image. To create one:

```
$ linuxkit build -format aws examples/aws.yml
```

## Push image

Before you do this you need to create a `vmimport` service role as explained in
[the VM import documentation](http://docs.aws.amazon.com/vm-import/latest/userguide/vmimport-image-import.html).

Do `linuxkit push aws -bucket bucketname aws.raw` to upload it to the
specified bucket, and create a bootable image from the stored image.

Alternatively, you can use the `AWS_BUCKET` environment variable to specify the bucket name.

**Note:** If the push times out before it finishes, you can use the `-timeout` flag to extend the timeout.

```
linuxkit push aws -bucket bucketname -timeout 1200 aws.raw
```

## Create an instance and connect to it

With the image created, we can now create an instance.
You won't be able to see the serial console output until after it has terminated.

```
linuxkit run aws aws
```

You can edit the AWS example to allow you to SSH to your instance in order to use it.
