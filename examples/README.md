Examples of building an image to run on LinuxKit or on a host

Currently the `moby` tool can output formats suitable for LinuxKit to boot on
a VM, and also some formats for running natively.

The `docker` format adds a `Dockerfile` to the tarball and expects a similar
file structure to `LinuxKit` but with the low level system setup pieces removed
as this will already be set up in a container.

The `mobytest/init-container` image in this repository has an example setup that
initialises `containerd` in exactly the same way as it runs in LinuxKit, which will
then start the `onboot` and `service` containers. The example below shows how you
can run `nginx` on either of these base configs.

```
moby build -format docker -o - docker.yml nginx.yml | docker build -t dockertest -
docker run -d -p 80:80 --privileged dockertest

moby build -format kernel+initrd linuxkit.yml nginx.yml
linuxkit run nginx
```

Both of these will run the same `nginx` either in a VM or a container.
