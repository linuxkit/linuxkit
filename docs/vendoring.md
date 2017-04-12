Vendoring
=========

The Go code in this repo depends on a number of Go libraries.
Theses are vendored in to the `vendor` directory using [`vndr`](https://github.com/lk4d4/vndr)
The `vendor.conf` file contains a list of the repositories and the git SHA or branch name that should be vendored

## Updating dependencies

Update `vendor.conf` with the dependency that you would like to add.
Details of usage of the `vndr` tool and the format of `vendor.conf` can be found [here](https://github.com/LK4D4/vndr/blob/master/README.md)

Once done, you must run the `vndr` tool to add the necessary files to the `vendor` directory.
The easiest way to do this is in a container.

## Updating in a container

To update all dependencies:

```
docker run -it --rm \
-v $(PWD):/go/src/github.com/docker/moby \
-w /go/src/github.com/docker/moby \
--entrypoint /go/bin/vndr \
linuxkit/go-compile:90607983001c2789911afabf420394d51f78ced8
```

To update a single dependency:

```
docker run -it --rm \
-v $(PWD):/go/src/github.com/docker/moby \
-w /go/src/github.com/docker/moby \
--entrypoint /go/bin/vndr \
linuxkit/go-compile:90607983001c2789911afabf420394d51f78ced8 \
github.com/docker/docker
```

## Updating locally

First you must install `vndr` and ensure that `$GOPATH/bin` is on your `$PATH`

```
go get -u github.com/LK4D4/vndr
```

To update all dependencies:

```
vndr
```

To update a single dependency:

```
vndr github.com/docker/docker
```
