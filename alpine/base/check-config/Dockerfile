FROM alpine:3.4
RUN apk update && apk upgrade && apk add --no-cache bash
ADD https://raw.githubusercontent.com/docker/docker/master/contrib/check-config.sh /usr/bin/check-config.sh
ENTRYPOINT ["/bin/bash", "/usr/bin/check-config.sh"]
