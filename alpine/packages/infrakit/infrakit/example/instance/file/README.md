InfraKit Instance Plugin - File
===============================

A [reference](../../../README.md#reference-implementations) implementation of an Instance Plugin that can accept any
configuration and writes the configuration to disk as `provision`.  It is useful for testing and debugging.

## Building

Begin by building plugin [binaries](../../../README.md#binaries).

## Usage

The plugin can be started without any arguments and will default to using unix socket in
`~/.infrakit/plugins` for communications with the CLI and other plugins:

```shell
$ build/infrakit-instance-file --dir=./test
INFO[0000] Listening at: ~/.infrakit/plugins/instance-file
```

This starts the plugin using `./test` as directory and `instance-file` as name.

You can give the another plugin instance a different name via the `listen` flag:
```shell
$ build/infrakit-instance-file --name=another-file --dir=./test
INFO[0000] Listening at: ~/.infrakit/plugins/another-file
```

Be sure to verify that the plugin is [discoverable](../../../cmd/cli/README.md#list-plugins).

Note that there should be two file instance plugins running now with different names
(`instance-file`, and `another-file`).

See the [CLI Doc](/cmd/cli/README.md) for details on accessing the instance plugin via CLI.
