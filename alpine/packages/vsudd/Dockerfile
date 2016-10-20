FROM mobylinux/alpine-build-go:78d9a136c632c66f0767e11ce3c661989d706f21

COPY ./ /go/src/vsudd/

WORKDIR /go/src/vsudd

RUN go install --ldflags '-extldflags "-fno-PIC"'

CMD ["tar", "cf", "-", "-C", "/go/bin", "vsudd"]
