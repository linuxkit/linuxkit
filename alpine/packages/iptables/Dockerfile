# Tag: alpine
FROM ocaml/opam@sha256:2d15235a8150d49353533848c8a2c326996558d57872acec59de35f8965dab4d
RUN sudo apk add m4
RUN opam install --use-internal-solver ocamlfind astring syslog -y
WORKDIR /app
ADD . /app
RUN sudo chown -R opam /app
RUN opam config exec -- ocamlfind ocamlopt -package unix,astring,syslog -linkpkg -o iptables main.ml
CMD ["tar", "cf", "-", "iptables"]
