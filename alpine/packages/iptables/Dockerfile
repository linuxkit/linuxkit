FROM mobylinux/alpine-build-ocaml
WORKDIR /app
ADD . /app
RUN sudo chown -R opam /app
RUN opam config exec -- ocamlfind ocamlopt -package unix,astring -linkpkg -o iptables main.ml
