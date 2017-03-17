FROM ocaml/opam:alpine-3.5_ocaml-4.04.0
RUN git -C /home/opam/opam-repository pull origin master && opam update -u

RUN opam pin add -n functoria https://github.com/samoht/functoria.git#output
RUN opam pin add -n mirage https://github.com/samoht/mirage.git#static
RUN opam pin add -n mirage-net-fd https://github.com/mirage/mirage-net-fd.git
RUN opam pin add -n charrua-client https://github.com/yomimono/charrua-client.git#state-halfway
RUN opam depext -uiy ocamlfind topkg-care ocamlbuild lwt mirage-types-lwt mirage
RUN opam depext -uiy charrua-client cohttp conduit mirage-unix
RUN opam depext -uiy mirage-net-fd ptime mirage-logs

RUN sudo mkdir -p /src
RUN sudo chown -R opam /src
WORKDIR /src

COPY compile.sh /usr/bin/

ENTRYPOINT ["/usr/bin/compile.sh"]
