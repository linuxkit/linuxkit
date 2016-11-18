FROM alpine:3.4
RUN apk update && apk upgrade && apk add --no-cache bash
ADD https://raw.githubusercontent.com/justincormack/docker/228ee16df3d738efaf001309370b26561c49edd6/contrib/check-config.sh /usr/bin/check-config.sh
ENTRYPOINT ["/bin/bash", "/usr/bin/check-config.sh"]
