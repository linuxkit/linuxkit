FROM alpine:3.13

ARG TEST_RESULT=FAILED

RUN echo "printf \"Build-arg test $TEST_RESULT\\n\"" >> check.sh

ENTRYPOINT ["/bin/sh", "/check.sh"]
