FROM alpine:3.5

RUN apk update && apk add lua git bash

RUN git clone https://github.com/alpinelinux/aports.git

ADD license.sh packages.lua /usr/bin/

CMD license.sh
