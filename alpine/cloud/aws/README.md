# Compile Moby for AWS (Amazon Machine Image)

#### Requirements

To compile, the requirements are:

1. Must be working on a EC2 instance
2. Must have `docker` and `docker-compose` installed
3. Must have configured Amazon credentials on instances (`aws configure`)

(The build will mount `~/.aws` into the build container).

#### Building

To bake the AMI:

```console
$ make ami
```

Inside of the `alpine/` subdirectory of the main Moby repo.

This will:

1. Clean up any remaining artifacts of old AMI builds
2. Creates a new EBS volume and attaches it to the build instance
3. Formats and partitions the volume for installation of Linux
4. Sets up artifacts (`initrd.img` and `vmlinuz64`) inside the new partition for booting
5. Installs MBR to boot syslinux to the device
6. Takes snapshot of EBS volume with Moby installed
7. Turns the snapshot into an AMI

#### Testing

Once the AMI has been created a file, `aws/ami_id.out` will be written which
contains its ID.

You can boot a small AWS instance from this AMI using the `aws/run-instance.sh`
script.

There is no SSH available today, but inbound access on the Docker API should
work if you configure a proper security group and attach it to the instance.

For instance, allow inbound access on `:2375` and a command such as this from
your compiler instance should work to get a "root-like" shell:

```console
$ docker -H 172.31.2.176:2375 \
    run -ti \
    --privileged \
    --pid host \
    debian \
    nsenter -t 1 -m
```

Alternatively, you can also have the `aws/run-instance.sh` script create a
security group and Swarm for you automatically (including worker/agent
instances to join the cluster).

To do so, set the `JOIN_INSTANCES` environment variable to any value, and
specify how many "joiners" (worker nodes) you want to also spin up using the
`JOINERS_COUNT` environment variable (the default is 1). e.g.:

```
$ JOIN_INSTANCES=1 JOINERS_COUNT=3 ./aws/run-instance.sh
```

This will give you a 4 node cluster with a manager named
`docker-swarm-manager`, and workers named `docker-swarm-joiner-0`,
`docker-swarm-joiner-1`, and so on.
