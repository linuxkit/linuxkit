### Capnp compiler

FROM alpine:3.5 as capnp

RUN mkdir -p /src
RUN apk update && apk add autoconf automake libtool linux-headers git g++ make

RUN cd /src && git clone -b v0.6.0 https://github.com/sandstorm-io/capnproto.git
WORKDIR /src/capnproto/c++
RUN ./setup-autotools.sh
RUN autoreconf -i
RUN ./configure
RUN make -j6 check
RUN make install
RUN which capnp


### SDK

FROM ocaml/opam:alpine-3.5_ocaml-4.04.0 as sdk
RUN git -C /home/opam/opam-repository pull && opam update -u

COPY --from=capnp /usr/local/bin/capnp /usr/local/bin/
COPY --from=capnp /usr/local/lib/libcapnpc-0.6.0.so /usr/local/lib/
COPY --from=capnp /usr/local/lib/libcapnp-0.6.0.so /usr/local/lib/
COPY --from=capnp /usr/local/lib/libkj-0.6.0.so /usr/local/lib/

RUN sudo mkdir -p /src
USER opam
WORKDIR /src

RUN opam pin add jbuilder 1.0+beta9 -n

RUN opam depext -uiy cstruct cstruct-lwt lwt lwt logs irmin-git rawlink tuntap astring rresult \
    mirage-flow-lwt mirage-channel-lwt io-page decompress capnp

RUN opam list

COPY ./sdk /src/
COPY ./sdk.opam /src/
RUN sudo chown opam -R /src

RUN opam config exec -- jbuilder build -p sdk @install
RUN opam config exec -- jbuilder install sdk

### Privileged Container

FROM sdk as priv

RUN opam pin add tuntap 1.0.0 -n
RUN opam depext -iy bos cmdliner

RUN opam list

COPY ./dhcp-client /src/dhcp-client
RUN sudo chown opam -R /src

RUN opam config exec -- jbuilder build dhcp-client/main.exe
RUN sudo mkdir -p /bin
RUN sudo cp /src/_build/default/dhcp-client/main.exe /bin/dhcp-client


### Calf Container

FROM sdk as calf

RUN opam pin add charrua-client https://github.com/yomimono/charrua-client.git#state-halfway -n
RUN opam pin add mirage-net-fd 0.2.0 -n
RUN opam depext -iy mirage-net-fd charrua-client lwt mirage-types-lwt cmdliner

RUN opam list

COPY ./dhcp-client-calf/unikernel.ml /src/dhcp-client-calf/
COPY ./dhcp-client-calf/jbuild /src/dhcp-client-calf/
RUN sudo chown opam -R /src

RUN opam config exec -- jbuilder build dhcp-client-calf/unikernel.exe
RUN sudo mkdir -p /bin/calf
RUN sudo cp /src/_build/default/dhcp-client-calf/unikernel.exe /bin/calf/dhcp-client-calf

### Final build

FROM scratch

COPY --from=priv /bin /
COPY --from=calf /bin /
COPY dhcp-client-calf/config.json /calf/
CMD ["/dhcp-client", "-vv"]
