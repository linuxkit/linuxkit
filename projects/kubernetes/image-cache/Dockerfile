FROM linuxkit/docker-ce:9e125aa533108731d11f6d7ec17aba6ded0cb4eb
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
