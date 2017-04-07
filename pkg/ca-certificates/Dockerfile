FROM debian:testing

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get -yq upgrade && apt-get install -yq ca-certificates

RUN printf "FROM scratch\nCOPY /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/\n" > Dockerfile
CMD ["tar", "cf", "-", "Dockerfile", "etc/ssl/certs/ca-certificates.crt"]
