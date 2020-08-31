FROM library/alpine:3.11
ARG MYARG=default

RUN echo "My var is ${MYARG}"

COPY test.sh /test.sh
CMD /test.sh
