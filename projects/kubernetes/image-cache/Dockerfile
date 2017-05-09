FROM linuxkit/docker-ce:050e734489f2d19b42ec818a4242a318ea446bc3
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
