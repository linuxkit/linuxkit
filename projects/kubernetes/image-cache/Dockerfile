FROM linuxkit/docker-ce:668d62da6e3da081a8f8aca7db3e2a98adf5da59
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
