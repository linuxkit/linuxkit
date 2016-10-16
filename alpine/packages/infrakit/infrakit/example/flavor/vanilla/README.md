InfraKit Flavor Plugin - Vanilla
================================

A [reference](../../../README.md#reference-implementations) implementation of a Flavor Plugin that supports direct
injection of Instance fields.

While we can specify a list of logical ID's (for example, IP addresses), `Init` and `Tags`
are all statically defined in the config JSON.  This means all the members of the group are
considered identical.

You can name your cattle but they are still cattle.  Pets, however, would imply strong identity
*as well as* special handling.  This is done via the behavior provided by the `Prepare` method of
the plugin.  This plugin applies the static configuration.


## Schema

Here's a skeleton of this Plugin's schema:
```json
{
  "Init": [],
  "Tags": {}
}
```

The supported fields are:
* `UserData`: an array of shell code lines to use for the Instance's Init script
* `Labels`: a string-string mapping of keys and values to add as Instance Tags

Here's an example Group configuration using the default [infrakit/group](/cmd/group) Plugin and the Vanilla Plugin:
```json
{
  "ID": "cattle",
  "Properties": {
    "Allocation": {
      "Size": 5
    },
    "Instance": {
      "Plugin": "instance-vagrant",
      "Properties": {
        "Box": "bento/ubuntu-16.04"
      }
    },
    "Flavor": {
      "Plugin": "flavor-vanilla",
      "Properties": {
        "Init": [
          "sudo apt-get update -y",
          "sudo apt-get install -y nginx",
          "sudo service nginx start"
        ],
        "Tags": {
            "tier": "web",
            "project": "infrakit"
        }
      }
    }
  }
}
```

Or with assigned IDs:
```json
{
  "ID": "named-cattle",
  "Properties": {
    "Allocation": {
      "LogicalIDs": [
        "192.168.0.1",
        "192.168.0.2",
        "192.168.0.3",
        "192.168.0.4",
        "192.168.0.5"
      ]
    },
    "Instance": {
      "Plugin": "instance-vagrant",
      "Properties": {
        "Box": "bento/ubuntu-16.04"
      }
    },
    "Flavor": {
      "Plugin": "flavor-vanilla",
      "Properties": {
        "Init": [
          "sudo apt-get update -y",
          "sudo apt-get install -y nginx",
          "sudo service nginx start"
        ],
        "Tags": {
          "tier": "web",
          "project": "infrakit"
        }
      }
    }
  }
}
```


## Example

Begin by building plugin [binaries](../../../README.md#binaries).

This plugin will be called whenever you use a Flavor plugin and reference the plugin by name
in your config JSON.  For instance, you may start up this plugin as `french-vanilla`:

```shell
$ build/infrakit-flavor-vanilla --name french-vanilla
INFO[0000] Listening at: ~/.infrakit/plugins/french-vanilla 
```

Then in your JSON config for the default group plugin, you would reference it by name:

```json
{
  "ID": "cattle",
  "Properties": {
    "Allocation": {
      "Size": 5
    },
    "Instance": {
      "Plugin": "instance-file",
      "Properties": {
        "Note": "Here is a property that only the instance plugin cares about"
      }
    },
    "Flavor": {
      "Plugin": "french-vanilla",
      "Properties": {
        "Init": [
          "sudo apt-get update -y",
          "sudo apt-get install -y nginx",
          "sudo service nginx start"
        ],
        "Tags": {
          "tier": "web",
          "project": "infrakit"
        }
      }
    }
  }
}
```
Then when you watch a group with the configuration above (`cattle`), the cattle will be `french-vanilla` flavored.
