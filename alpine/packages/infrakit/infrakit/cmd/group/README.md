InfraKit Group Plugin
=====================

This is the default implementation of the Group Plugin that can manage collections of resources.
This plugin works in conjunction with the Instance and Flavor plugins, which separately define
the properties of the physical resource (Instance plugin) and semantics or nature  of the node
(Flavor plugin).


## Running

Begin by building plugin [binaries](../../README.md#binaries).

The plugin may be started without any arguments and will default to using unix socket in
`~/.infrakit/plugins` for communications with the CLI and other plugins:

```shell
$ build/infrakit-group-default
INFO[0000] Listening at: ~/.infrakit/plugins/group
```
