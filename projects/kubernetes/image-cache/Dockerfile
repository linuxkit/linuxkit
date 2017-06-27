FROM linuxkit/docker-ce:530912564c0b648aeeab2459c7b82ce40d48fd6a
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
