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
linuxkit/go-compile:7b1f5a37d2a93cd4a9aa2a87db264d8145944006
mod vendor
```
