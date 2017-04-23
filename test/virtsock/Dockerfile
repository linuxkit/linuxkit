FROM linuxkit/go-compile:4513068d9a7e919e4ec42e2d7ee879ff5b95b7f5@sha256:bdfadbe3e4ec699ca45b67453662321ec270f2d1a1dbdbf09625776d3ebd68c5 AS virtsock-build

ENV VIRTSOCK_COMMIT=6b4dec728264e07c41e108caebd6bc2b72559a5f
RUN mkdir -p $GOPATH/src/github.com/rneugeba && \
  cd $GOPATH/src/github.com/rneugeba && \
  git clone https://github.com/rneugeba/virtsock.git
WORKDIR $GOPATH/src/github.com/rneugeba/virtsock
RUN git checkout $VIRTSOCK_COMMIT
RUN make build/virtsock_stress.linux
RUN cp -a build/virtsock_stress.linux /virtsock_stress

FROM linuxkit/tini:6714d66b82b5397f497b2aa05764096ed1ffe7d7@sha256:ba594b96af6195737ce2df702196d7adea2cafde554e18940ee14ad575d27f3b
COPY --from=virtsock-build virtsock_stress bin/virtsock_stress
CMD ["/bin/tini", "/bin/virtsock_stress", "-s", "-v", "1"]
