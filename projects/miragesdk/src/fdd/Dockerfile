FROM ocaml/opam:alpine as base
RUN sudo apk add m4
RUN opam install jbuilder lwt fd-send-recv logs fmt cmdliner astring
ADD . /src
RUN opam pin add fdd /src
RUN sudo mkdir -p /out/bin
RUN sudo cp /home/opam/.opam/4.04.2/bin/fdd /out/bin

FROM scratch
COPY --from=base /out .
USER 0
ENTRYPOINT ["/fdd"]
