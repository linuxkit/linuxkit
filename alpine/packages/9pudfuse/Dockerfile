FROM ocaml/opam:alpine-3.3_ocaml-4.02.3

ENV \
  OPAMYES=true \
  CAML_LD_LIBRARY_PATH="/root/.opam/4.02.3/lib/stublibs:/usr/pkg/lib/ocaml/stublibs" \
  MANPATH="/root/.opam/4.02.3/man:" \
  PERL5LIB="/root/.opam/4.02.3/lib/perl5" \
  OCAML_TOPLEVEL_PATH="/root/.opam/4.02.3/lib/toplevel" \
  PATH=/root/.opam/4.02.3/bin:$PATH

COPY . .
RUN opam depext -i result stringext fd-send-recv
RUN opam config exec -- ocamlbuild -tag debug -use-ocamlfind -package stringext,fd-send-recv,result fuse_socket_9p.native
