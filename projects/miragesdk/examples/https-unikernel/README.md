## https-unikernel example service

This sample service implements an https web-server as a set of three components communicating using [Cap'n Proto RPC][].

1. The `store` service accepts requests for web pages and returns the page content.
2. The `http` service accepts HTTP connections over RPC and handles GET requests using `store`.
3. The `tls` service accepts encrypted HTTPS connections and provides them as a plain-text stream to `http`.

The protocols implemented by the services can be found in the `proto.capnp` file.

The services can be run in separate processes so that they are isolated from each other.
For example, only the `tls` component needs access to the private key, so
a bug in the HTTP protocol decoder cannot leak the key.

Although the example services are all written in OCaml, it should be possible to replace any of them with a different implementation written in any other language with Cap'n Proto RPC support.

### Running the samples

The easiest way to build and run is using Docker:

    make docker
    make docker-run

Once inside the container, you can run all the services in a single process like this:

    https-unikernel-single

You should be able to try out the service by opening <https://localhost:8443> in a browser.

You can also run the service as three separate processes.
The easiest way to do this is by creating multiple windows in `screen` (which is running by default in the Docker image).

First, start the store:

    https-unikernel-store unix:/tmp/store.sock

Then, create a new window (`Ctrl-A c`) and start the http service:

    https-unikernel-http unix:/tmp/http.sock --store=unix:/tmp/store.sock

Finally, make another window and run the TLS terminator:

    https-unikernel-tls --http unix:/tmp/http.sock --port 8443


### Testing with Python

With the services running as separate processes (i.e. with socket files available), you can also invoke services from other languages.
For example, using Python (make another screen split):

```
$ cd src
$ python
>>> import capnp
>>> import proto_capnp
>>> import socket
>>> s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM, 0)
>>> s.connect('/tmp/store.sock')
>>> client = capnp.TwoPartyClient(s)
>>> store = client.bootstrap().cast_as(proto_capnp.Store)
>>> r = store.get(['index.html'])
>>> print r.wait()
( ok = "<p>It works!</p><p>Powered by Irmin.</p>\n" )
```

(or just do `python src/test_store.py`)


[Cap'n Proto RPC]: https://capnproto.org/rpc.html
