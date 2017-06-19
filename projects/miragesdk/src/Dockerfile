### Capnp compiler

FROM alpine:3.5 as capnp

RUN mkdir -p /src
RUN apk update && apk add autoconf automake libtool linux-headers git g++ make

RUN cd /src && git clone https://github.com/sandstorm-io/capnproto.git
WORKDIR /src/capnproto/c++
RUN ./setup-autotools.sh
RUN autoreconf -i
RUN ./configure
RUN make -j6 check
RUN make install
RUN which capnp


### SDK

FROM ocaml/opam@sha256:1e1d7fafbfd461bf684b5e11213c85a71fec93577455285e5d82d902ffad91d2 as sdk
#FROM ocaml/opam:alpine-3.5_ocaml-4.04.0 as sdk

COPY --from=capnp /usr/local/bin/capnp /usr/local/bin/
COPY --from=capnp /usr/local/bin/capnpc /usr/local/bin/
COPY --from=capnp /usr/local/lib/libcapnpc-0.7-dev.so /usr/local/lib/
COPY --from=capnp /usr/local/lib/libcapnp-0.7-dev.so /usr/local/lib/
COPY --from=capnp /usr/local/lib/libkj-0.7-dev.so /usr/local/lib/

RUN sudo mkdir -p /src
USER opam
WORKDIR /src

RUN git -C /home/opam/opam-repository fetch origin && \
    git -C /home/opam/opam-repository reset ad921dfa87c2e201ef54806d0367aaacce75bc62 --hard && \
    opam update -u

RUN opam pin add -n capnp.dev 'https://github.com/talex5/capnp-ocaml.git#interfaces'
RUN opam pin add -n capnp-rpc.dev 'https://github.com/mirage/capnp-rpc.git'
RUN opam pin add -n capnp-rpc-lwt.dev 'https://github.com/mirage/capnp-rpc.git'
RUN opam pin add -n charrua-client.dev https://github.com/samoht/charrua-client.git#with-cdhcpc
RUN opam pin add -n mirage-flow-rawlink.dev https://github.com/mirage/mirage-flow-rawlink.git
RUN opam pin add -n mirage-flow-lwt --dev
RUN opam pin add -n mirage-flow-unix --dev

COPY sdk.opam /src
RUN sudo chown opam -R /src
RUN opam pin add sdk.local /src -n

RUN opam depext -y alcotest sdk
RUN opam install alcotest && opam install --deps sdk

RUN opam list

COPY ./sdk /src/
RUN sudo chown opam -R /src

RUN opam update sdk && opam install sdk -t

### Privileged Container

FROM sdk as priv

COPY ./dhcp-client /src/dhcp-client
RUN sudo chown opam -R /src

RUN opam config exec -- jbuilder build --dev dhcp-client/main.exe
RUN sudo mkdir -p /out
RUN sudo cp /src/_build/default/dhcp-client/main.exe /out/dhcp-client


### Calf Container

FROM sdk as calf

COPY ./dhcp-client-calf/unikernel.ml /src/dhcp-client-calf/
COPY ./dhcp-client-calf/jbuild /src/dhcp-client-calf/
RUN sudo chown opam -R /src

RUN opam config exec -- jbuilder build --dev dhcp-client-calf/unikernel.exe
RUN sudo mkdir -p /out/
RUN sudo cp /src/_build/default/dhcp-client-calf/unikernel.exe /out/dhcp-client-calf

### Final build

FROM scratch

USER 0
COPY --from=priv /out /
COPY --from=calf /out /calf/rootfs/
COPY dhcp-client-calf/config.json /calf
CMD ["/dhcp-client", "-vv"]
