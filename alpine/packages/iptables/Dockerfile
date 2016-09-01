FROM ocaml/opam:alpine
RUN sudo apk add m4
RUN opam install ocamlfind astring syslog -y
WORKDIR /app
ADD . /app
RUN sudo chown -R opam /app
RUN opam config exec -- ocamlfind ocamlopt -package unix,astring,syslog -linkpkg -o iptables main.ml
