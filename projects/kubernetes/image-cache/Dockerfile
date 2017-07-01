FROM linuxkit/docker-ce:9b937df179bdbebbc70243779978057df0b54190
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
