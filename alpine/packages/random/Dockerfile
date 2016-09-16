FROM mobylinux/alpine-build-c:1ea690a7438cdd8a5965eaa034e0a0c521d9cb40

COPY . /random

WORKDIR /random

RUN cc -g -static -Wall -Werror -o add-random rdrand.c rdseed.c main.c

CMD ["tar", "cf", "-", "add-random"]
