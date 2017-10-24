Vendoring
=========

The Go code in this repo depends on a number of Go libraries.
These are vendored in to the `src/cmd/linuxkit/vendor` directory using [`vndr`](https://github.com/lk4d4/vndr)
The `vendor.conf` file contains a list of the repositories and the git SHA or branch name that should be vendored

## Updating dependencies

Update `src/cmd/linuxkit/vendor.conf` with the dependency that you would like to add.
Details of usage of the `vndr` tool and the format of `vendor.conf` can be found [here](https://github.com/LK4D4/vndr/blob/master/README.md)

Once done, you must run the `vndr` tool to add the necessary files to the `vendor` directory.
The easiest way to do this is in a container.

## Updating in a container

To update all dependencies:

```
docker run -it --rm \
-v $(pwd):/go/src/github.com/linuxkit/linuxkit \
-w /go/src/github.com/linuxkit/linuxkit/src/cmd/linuxkit \
--entrypoint /go/bin/vndr \
linuxkit/go-compile:fb53f01a669de5e91ec855b4f67a57b514b4f6ed
```

To update a single dependency:

```
docker run -it --rm \
-v $(pwd):/go/src/github.com/linuxkit/linuxkit \
-w /go/src/github.com/linuxkit/linuxkit/src/cmd/linuxkit \
--entrypoint /go/bin/vndr \
linuxkit/go-compile:fb53f01a669de5e91ec855b4f67a57b514b4f6ed
github.com/docker/docker
```

## Updating locally

First you must install `vndr` and ensure that `$GOPATH/bin` is on your `$PATH`

```
go get -u github.com/LK4D4/vndr
```

To update all dependencies:

```
cd src/cmd/linuxkit
vndr
```

To update a single dependency:

```
cd /src/cmd/linuxkit
vndr github.com/docker/docker
```
