FROM ocaml/opam:alpine
RUN sudo apk update && sudo apk upgrade
RUN opam update -u
RUN opam depext -uiy topkg-care ocamlbuild lwt

RUN opam remote add mirage-dev https://github.com/mirage/mirageos-3-beta.git
RUN opam pin add mirage-net-fd https://github.com/mirage/mirage-net-fd.git
RUN opam depext -uiy mirage tcpip charrua-client

RUN sudo mkdir -p /src/charrua
RUN sudo chown -R opam /src
WORKDIR /src/charrua

COPY compile.sh /usr/bin/

ENTRYPOINT ["/usr/bin/compile.sh"]
