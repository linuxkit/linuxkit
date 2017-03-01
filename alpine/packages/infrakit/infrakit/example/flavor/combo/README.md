InfraKit Flavor Plugin - Combo
==============================

A [reference](../../../README.md#reference-implementations) implementation of a Flavor Plugin that supports composition
of other Flavors.

The Combo plugin allows you to use Flavors as mixins, combining their Instance properties:
  * `Tags`: combined, with any colliding values determined by the last Plugin to set them
  * `Init`: concatenated in the order of the configuration, separated by a newline
  * `Attachments`: combined in the order of the configuration

## Schema

Here's a skeleton of this Plugin's schema:
```json
{
  "Flavors": []
}
```

A single field, `Flavors`, is supported, which is an array of the Flavors to compose.  Each element in the array is the
same structure as how Flavors are used elsewhere:

```json
{
  "Plugin": "",
  "Properties": {
  }
}
```


## Example

To demonstrate how the Combo Flavor plugin works, we will compose two uses of the Vanilla plugin together.

First, start up the plugins we will use:

```shell
$ build/infrakit-group-default
INFO[0000] Listening at: ~/.infrakit/plugins/group
```

```shell
$ mkdir -p tutorial
$ build/infrakit-instance-file --dir tutorial
INFO[0000] Listening at: ~/.infrakit/plugins/instance-file
```

```shell
$ build/infrakit-flavor-vanilla
INFO[0000] Listening at: ~/.infrakit/plugins/flavor-vanilla
```

```shell
$ build/infrakit-flavor-combo
INFO[0000] Listening at: ~/.infrakit/plugins/flavor-combo
```

Using the [example](example.json) configuration, start watching a group:
```shell
$ build/infrakit group watch example/flavor/combo/example.json
watching combo
```

You will notice that the configuration is somewhat nonsensical, as the result could have been achieved without
using the Combo plugin.  However, it illustrates how the two plugin properties are combined to form the instance
properties. Specifically, note how both `Tags` and both `Init` lines are present:
```shell
$ cat tutorial/instance-4039631736808433938
{
    "ID": "instance-4039631736808433938",
    "LogicalID": null,
    "Tags": {
      "infrakit.config_sha": "pAD2EkxjoqO35Dx5UZUIehOU-Go=",
      "infrakit.group": "combo",
      "v1": "tag one",
      "v2": "tag two"
    },
    "Spec": {
      "Properties": {},
      "Tags": {
        "infrakit.config_sha": "pAD2EkxjoqO35Dx5UZUIehOU-Go=",
        "infrakit.group": "combo",
        "v1": "tag one",
        "v2": "tag two"
      },
      "Init": "vanilla one\nvanilla two",
      "LogicalID": null,
      "Attachments": null
    }
  }
```

