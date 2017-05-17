FROM alpine:edge

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 17.05.0-ce
ENV DOCKER_SHA256 340e0b5a009ba70e1b644136b94d13824db0aeb52e09071410f35a95d94316d9

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  curl \
  openssh-server \
  tini \
  util-linux \
  && true

RUN set -x \
	&& curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-${DOCKER_VERSION}.tgz" -o docker.tgz \
	&& echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
	&& tar -xzvf docker.tgz \
	&& mv docker/* /usr/bin/ \
	&& rmdir docker \
	&& rm docker.tgz \
	&& docker -v

COPY . .

RUN mkdir -p /etc/ssh /root/.ssh && chmod 0700 /root/.ssh

CMD ["/sbin/tini", "/usr/bin/ssh.sh"]
