FROM docker:17.06.0-ce-dind
COPY *.tar /images/
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
LABEL org.mobyproject.config='{"binds": ["/var/run:/var/run"]}'
