FROM ocaml/opam@sha256:e2e0dbbc859e078504d3a084feda27194406badf0d3d7e3d5321179c1c03190b
#FROM ocaml/opam:debian-9_ocaml-4.04.0
RUN cd opam-repository && git fetch && git reset --hard df060ffa5c9d62ec63395fa80d0f5b50a5863c47 && opam update
RUN opam depext -i -y jbuilder lwt cohttp astring tls capnp camlzip alcotest cohttp capnp-rpc-unix
RUN sudo apt-get install -y screen python-pip python-setuptools python-dev --no-install-recommends
RUN pip install cython pycapnp
ADD opam /home/opam/src/opam
RUN opam pin add -ny mypkg /home/opam/src
RUN opam install --deps-only mypkg
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
