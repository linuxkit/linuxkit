Base repo for Moby, codename for the Docker Linux distro

Initial requirements are being driven by the very minimal goal of replacing boot2docker for the new Mac app.

However these requirements are fairly small and the scope is intended to be much broader.

Build instructions: use `make` to build. `make xhyve` will boot it up on a Mac; unless you run with `sudo` you will not get any networking.
