FROM ubuntu:15.10

RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get -y install build-essential automake pkg-config libtool gnu-efi

RUN mkdir -p /gummiboot

COPY gummiboot /gummiboot
RUN tar czvf /gummiboot.tar.gz gummiboot

WORKDIR /gummiboot

RUN ./autogen.sh && \
    ./configure && \
    make linuxx64.efi.stub
