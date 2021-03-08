Vendoring
=========

The Go code in this repo depends on a number of Go libraries.
These are vendored in to the `src/cmd/linuxkit/vendor` directory using [go modules](https://golang.org/ref/mod)

## Updating dependencies

Go modules should install any required dependencies to `go.mod` and `go.sum` when running normal go commands such as `go build`,
`go vet`, etc. To install specific versions, use `go get <dependency>@<reference>`.

See the [go modules](https://golang.org/ref/mod) documentation for more information.

LinuxKit vendors all dependencies to make it completely self-contained. Once `go.mod` is up to date,
you must update the dependencies, either using your local go toolchain or in a container.

## Updating locally

To vendor all dependencies:

1. `cd src/cmd/linuxkit`
1. Run `go mod vendor`

## Updating in a container

To update all dependencies:

```
docker run -it --rm \
-v $(pwd):/go/src/github.com/linuxkit/linuxkit \
-w /go/src/github.com/linuxkit/linuxkit/src/cmd/linuxkit \
--entrypoint=go
linuxkit/go-compile:b1446b2ba407225011f97ae1dba0f512ae7f9b84
mod vendor
```
