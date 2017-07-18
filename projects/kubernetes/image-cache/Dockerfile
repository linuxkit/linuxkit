FROM docker:17.06.0-ce-dind
ADD . /images
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
LABEL org.mobyproject.config='{"binds": ["/var/run:/var/run"]}'
