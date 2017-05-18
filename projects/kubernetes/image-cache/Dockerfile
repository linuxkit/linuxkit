FROM linuxkit/docker-ce:261f93927d85001c65e5ce0f421eb6062f09c0a5
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
