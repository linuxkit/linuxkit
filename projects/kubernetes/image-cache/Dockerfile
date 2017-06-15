FROM linuxkit/docker-ce:dda71ff9fe5ebbfa794b98c57c32df286b212848
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
