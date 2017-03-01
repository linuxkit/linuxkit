## Hyperkit/Moby Infrakit plugin

This is a Hyper/Kit Moby instance plugin for infrakit. The instance
plugin is capable to start/manage several hyperkit instances with with
different configurations and Moby configurations.

The plugin keeps state in a local directory (default `./vms`) where
each instance keeps some state in a sub-directory. The VM state
directory can be specified at the kernel command line using the
`--vm-dir` option.

The plugin also needs to know where the kernel and `initrd` images are
located. The `--vm-lib` command line argument to the plugin specifies
the directory. Inside the VM library directory every kernel/initrd configuration should be stored in a sub-directory. The sub-directory name is used in the instance configuration.

## Building

```sh
make
```
(you need a working docker installation...testing on Docker for Mac)

## Quickstart

To play round with the plugin, simply follow the [infrakit tutorial](https://github.com/docker/infrakit/blob/master/docs/tutorial.md) and replace the file instance plugin with:
```
./build/infrakit-instance-hyperkit --vm-lib ./vmlib
```
where `./vmlib` contains a sub-directory named `default` with a `vmlinuz64` and `initrd.img` image.

Instead of the `cattle.json` in the infrakit tutorial, use `hyperkit.json` in this directory.
