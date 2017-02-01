FROM alpine:3.5

RUN \
  apk update && apk upgrade && \
  apk add \
  docker \
  tar \
  && true

COPY . .

ENTRYPOINT ["/mkimage.sh"]
