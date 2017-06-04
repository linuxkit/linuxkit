# Compose Project

The purpose of this project is to show how moby and linuxkit can be used to build a runnable linuxkit image with compose-style apps ready-to-run.

The apps are simple:

* nginx serving app A
* nginx serving app B
* traefik routing on the basis of hostname between apps A and B


## Compose Methods
We provide samples of two methods for using compose: dynamic and static.

Both methods use the image `linuxkit/compose`. The image does the following:

1. Wait for compose to be ready.
2. If there are any tar files available in `/images/*.tar`, treat them as tarred up docker images and load them into docker via `docker load ...`
3. Run `docker compose ...`

The only difference between dynamic and static is whether or not container images are pre-loaded in the linuxkit image.

* Compose: the `linuxkit/compose` image looks for a compose file at `/compose/docker-compose.yml`
* Images: the `linuxkit/compose` image looks for tarred container images at `/compose/images/*.tar`

### Dynamic
Dynamic loads the _compose_ config into the linuxkit image at build time. Container images are not pre-loaded, and thus docker loads the container images from the registry **at run-time**. This is no different than doing the following:

1. Using the docker linuxkit image
2. Connecting remotely via the docker API
3. Running the compose file remotely

Except that the compose is run at launch time, and there is no need for a remote connection to the docker API.

It works by loading the `docker-compose.yml` file onto the linuxkit image, and making it available to the `compose` container image via a bind-mount.

To build a dynamic image, do `make dynamic`. To run it, do `make run-dynamic`.

### Static
Static loads the _compose_ config **and** the container _images_ into the linuxkit image at build time. When run, docker loads the images from its local cache and does not depend on access to a registry.

It works by loading the `docker-compose.yml` file onto the linuxkit image and tarred up container image files. It then makes them available to the `compose` container image via bind-mounts.


To build a static image, do `make static`. To run it, do `make run-static`.

Static images pre-load them by doing:

1. Download the image locally `docker image pull <image>`
2. Save the image to a local tar file `docker image save -o <image> <imagename>.tar`
3. copy the tar file to the container image
4. When starting the container from the image with the files, load the images into docker before running compose: `docker image load -i <image> && rm -f <imagename>.tar`

### Conversion
A final option would be converting all of the containers defined in a `docker-compose.yml` into linuxkit `services`. It also would require setting up appropriate networks and other services provided by docker when running compose.

An example may be added in the future.
