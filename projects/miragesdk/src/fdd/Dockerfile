FROM ocaml/opam:alpine as base
RUN sudo apk add m4
RUN opam install jbuilder lwt fd-send-recv logs fmt cmdliner
ADD . /src
RUN opam pin add fdd /src
RUN sudo mkdir /out
RUN sudo cp /home/opam/.opam/4.04.2/bin/fdd /out

FROM scratch
COPY --from=base /out/fdd .
USER 0
ENTRYPOINT ["/fdd"]
