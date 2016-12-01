# Tag: alpine
FROM ocaml/opam@sha256:fe4f6783f319fd02829af4f5a154d8e7030c3907ef393a7b7f21552648f89db8
RUN sudo apk add m4
RUN opam install ocamlfind astring syslog -y
WORKDIR /app
ADD . /app
RUN sudo chown -R opam /app
RUN opam config exec -- ocamlfind ocamlopt -package unix,astring,syslog -linkpkg -o iptables main.ml
CMD ["tar", "cf", "-", "iptables"]
