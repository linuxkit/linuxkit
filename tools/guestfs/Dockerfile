FROM debian:stretch

# Copy Dockerfile so we can include it in the hash
COPY Dockerfile /Dockerfile

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get -yq upgrade && apt-get install -yq libguestfs-tools syslinux linux-image-amd64 vim
