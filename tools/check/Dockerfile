FROM alpine:3.5
RUN apk update && apk upgrade && apk add --no-cache bash
ADD https://raw.githubusercontent.com/docker/docker/master/contrib/check-config.sh /check-config.sh
ADD . ./
ENTRYPOINT ["/bin/sh", "/check.sh"]
