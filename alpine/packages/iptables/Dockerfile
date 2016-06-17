FROM ocaml/opam:alpine
RUN sudo apk add m4
RUN opam install ocamlfind astring -y
WORKDIR /app
ADD . /app
RUN sudo chown -R opam /app
RUN opam config exec -- ocamlfind ocamlopt -package unix,astring -linkpkg -o iptables main.ml
