FROM mobylinux/docker-ce:741bf21513328f674e0cdcaa55492b0b75974e08
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
