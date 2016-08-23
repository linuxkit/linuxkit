These Dockerfiles are the base images for intermediate containers that are used for builds.

The aim is that if you have all the containers pulled you should not need network access to build.

Unlike the mobylinux/alpine-base image we do not mind so much about exact reproducibility as these
do not ship in the finished product.

These are autobuilds on Hub.
