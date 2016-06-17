FROM ocaml/opam:alpine
WORKDIR /app
ADD . /app
RUN sudo chown -R opam /app
RUN ocamlopt -o iptables main.ml
