FROM alpine:3.17

ARG SOURCE=FAILED
ARG REVISION=FAILED
ARG GOPKGVERSION=FAILED

RUN echo "printf \"Build-arg test source $SOURCE\\n\"" >> check.sh
RUN echo "printf \"Build-arg test revision $REVISION\\n\"" >> check.sh
RUN echo "printf \"Build-arg test gopkgversion $GOPKGVERSION\\n\"" >> check.sh

ENTRYPOINT ["/bin/sh", "/check.sh"]
