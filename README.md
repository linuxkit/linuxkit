Moby, a toolkit for custom Linux distributions

Simple build instructions: use `make` to build.

`make qemu` will boot up a sample in qemu in a container; on OSX `make hyperkit` will
boot up in hyperkit. `make test` or `make hyperkit-test` will run the test suite.

Requires GNU `make`, GNU or BSD `tar` (not Busybox tar) and Docker to build.

To customise, copy or modify the `moby.yaml` and then run `./bin/moby file.yaml` to
generate. You can run the output with `./scripts/qemu.sh` or `./scripts/hyperkit.sh`.
