# Kubernetes end-to-end test suite (e2e)

In this subdirectory you can build Kubernetes e2e image, that you can
use with `docker run` on Desktop or bundle as a job to run on a cluster
elsewhere.

## Building the image

```
make build HASH=current
```

This will result in `linuxkitprojects/kubernetes-e2e:current` image that
you can use. See `e2e.sh` for supported environment variables.

## Running on Desktop

```
make local
```

## Test results

The image is configuted to print test results to standard output.
Please consult [Kubernetes documentation for more information][e2e-docs].

[e2e-docs]: https://github.com/kubernetes/community/blob/master/contributors/devel/e2e-tests.md
