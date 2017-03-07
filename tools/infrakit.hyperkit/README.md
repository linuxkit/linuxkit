## Hyperkit/Moby Infrakit plugin

This is a Hyper/Kit Moby instance plugin for infrakit. The instance
plugin is capable to start/manage several hyperkit instances with with
different configurations and Moby configurations.

The plugin keeps state in a local directory (default
`.infrakit/hyperkit-vms`) where each instance keeps some state in a
sub-directory. The VM state directory can be specified at the kernel
command line using the `--vm-dir` option.

## Building

```sh
make
```
(you need a working docker installation, such as Docker for Mac)

## Quickstart

This is roughly based on the [infrakit tutorial](https://github.com/docker/infrakit/blob/master/docs/tutorial.md). You need to have the infrakit binaries in your path (or adjust the invocation of the commands below).  To get the binaries, it's best to compile from source (checkout `https://github.com/docker/infrakit.git`, then `make` or `make build-in-container`). The add the `./build` directory to your path.

Start the default group plugin:
```shell
infrakit-group-default
```
and the vanilla flavour plugin:
```shell
infrakit-flavor-vanilla
```

Then start the hyperkit plugin:
```shell
./build/infrakit-instance-hyperkit
```

Next, you can commit a new configuration. There is a sample infrakit config file in `hyperkit.json` which assumes that the directory `./vms/default` contains the `vmlinuz64` and `initrd.img` image to boot.
```
infrakit group commit hyperkit.json
```

This will create a single hyperkit instance with its state stored in
`~/.infrakit/hyperkit-vms`. There is a `tty` file which you can
connect to with `screen` to access the VM.

If you kill the hyperkit process a new instance will be restarted. If
you change the VM parameter in JSON file and commit the new config, a
new VM will be created. f you change the `Size` parameter, multiple
VMs will get started.

