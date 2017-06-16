FROM ocaml/opam@sha256:a469435632d0cacbceab799d7e48201b727d025fa1805cbbe210d94233b251ad
#FROM ocaml/opam:debian-9_ocaml-4.04.0
RUN opam pin add -yn capnp 'https://github.com/talex5/capnp-ocaml.git#demo1'
RUN opam pin add -yn capnp-rpc 'https://github.com/talex5/capnp-rpc.git#demo1'
RUN opam pin add -yn capnp-rpc-lwt 'https://github.com/talex5/capnp-rpc.git#demo1'
RUN opam depext -i -y jbuilder lwt cohttp astring tls capnp camlzip alcotest cohttp
RUN sudo apt-get install -y screen python-pip python-setuptools python-dev --no-install-recommends
RUN pip install cython pycapnp
ADD opam /home/opam/src/opam
RUN opam pin add -ny mypkg /home/opam/src
RUN opam install --deps-only mypkg
ENV JBUILD_STATIC=true
WORKDIR /home/opam/src
ADD . /home/opam/src
RUN sudo chown -R opam .
RUN opam config exec -- make
USER root
RUN cp _build/default/src/main.exe /usr/bin/https-unikernel-single && \
    cp _build/default/src/store_main.exe /usr/bin/https-unikernel-store && \
    cp _build/default/src/http_main.exe /usr/bin/https-unikernel-http && \
    cp _build/default/src/tls_main.exe /usr/bin/https-unikernel-tls
USER opam
ENV SHELL=bash
