# A Quick Tutorial

To illustrate the concept of working with Group, Flavor, and Instance plugins, we use a simple setup composed of
  + The default `group` plugin - to manage a collection of instances
  + The `file` instance plugin - to provision instances by writing files to disk
  + The `vanilla` flavor plugin - to provide context/ flavor to the configuration of the instances

It may be helpful to familiarize yourself with [plugin discovery](../README.md#plugin-discovery) if you have not already
done so.

Start the default Group plugin

```shell
$ build/infrakit-group-default
INFO[0000] Listening at: ~/.infrakit/plugins/group
```

Start the file Instance plugin

```shell
$ mkdir -p tutorial
$ build/infrakit-instance-file --dir ./tutorial/
INFO[0000] Listening at: ~/.infrakit/plugins/instance-file
```
Note the directory `./tutorial` where the plugin will store the instances as they are provisioned.
We can look at the files here to see what's being created and how they are configured.

Start the vanilla Flavor plugin

```shell
$ build/infrakit-flavor-vanilla
INFO[0000] Listening at: ~/.infrakit/plugins/flavor-vanilla
```

Show the plugins:

```shell
$ build/infrakit plugin ls
Plugins:
NAME                    LISTEN
flavor-vanilla          ~/.infrakit/plugins/flavor-vanilla
group                   ~/.infrakit/plugins/group
instance-file           ~/.infrakit/plugins/instance-file
```

Note the names of the plugin.  We will use the names in the `--name` flag of the plugin CLI to refer to them.

Here we have a configuration JSON for the group.  In general, the JSON structures follow a pattern:

```json
{
   "Plugin": "PluginName",
   "Properties": {
   }
}
```

This defines the name of the `Plugin` to use and the `Properties` to configure it with.  The plugins are free to define
their own configuration schema.  Plugins in this repository follow a convention of using a `Spec` Go struct to define
the `Properties` schema for each plugin.  The [`group.Spec`](/plugin/group/types/types.go) in the default Group plugin,
and [`vanilla.Spec`](/plugin/flavor/vanilla/flavor.go) are examples of this pattern.

From listing the plugins earlier, we have two plugins running. `instance-file` is the name of the File Instance Plugin,
and `flavor-vanilla` is the name of the Vanilla Flavor Plugin.
So now we have the names of the plugins and their configurations.

Putting everything together, we have the configuration to give to the default Group plugin:

```shell
$ cat << EOF > cattle.json
{
  "ID": "cattle",
  "Properties": {
    "Allocation": {
      "Size": 5
    },
    "Instance": {
      "Plugin": "instance-file",
      "Properties": {
        "Note": "Instance properties version 1.0"
      }
    },
    "Flavor": {
      "Plugin": "flavor-vanilla",
      "Properties": {
        "Init": [
          "docker pull nginx:alpine",
          "docker run -d -p 80:80 nginx-alpine"
        ],
        "Tags": {
          "tier": "web",
          "project": "infrakit"
        }
      }
    }
  }
}
EOF
```

Note that we specify the number of instances via the `Size` parameter in the `flavor-vanilla` plugin.  It's possible
that a specialized Flavor plugin doesn't even accept a size for the group, but rather computes the optimal size based on
some criteria.

Checking for the instances via the CLI:

```shell
$ build/infrakit instance --name instance-file describe
ID                              LOGICAL                         TAGS

```

Let's tell the group plugin to `watch` our group by providing the group plugin with the configuration:

```shell
$ build/infrakit group watch cattle.json
watching cattle
```

The group plugin is responsible for ensuring that the infrastructure state matches with your specifications.  Since we
started out with nothing, it will create 5 instances and maintain that state by monitoring the instances:
```shell
$ build/infrakit group inspect cattle
ID                              LOGICAL         TAGS
instance-1475104926           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104936           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104946           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104956           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104966           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
```

The Instance Plugin can also report instances, it will report all instances across all groups (not just `cattle`).

```shell
$ build/infrakit instance --name instance-file describe
ID                              LOGICAL         TAGS
instance-1475104926           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104936           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104946           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104956           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475104966           	  -             infrakit.config_sha=Y23cKqyRpkQ_M60vIq7CufFmQWk=,infrakit.group=cattle,project=infrakit,tier=web
```

Now let's update the configuration by changing the size of the group and a property of the instance:

```shell
$ diff cattle.json cattle2.json 
7c7
<                 "Note": "Instance properties version 1.0"
---
>                 "Note": "Instance properties version 2.0"
13c13
<                 "Size": 5,
---
>                 "Size": 10,
```

Before we do an update, we can see what the proposed changes are:
```shell
$ build/infrakit group describe cattle2.json 
Performs a rolling update on 5 instances, then adds 5 instances to increase the group size to 10
```

So here 5 instances will be updated via rolling update, while 5 new instances at the new configuration will
be created.

Let's apply the new config:

```shell
$ build/infrakit group update cattle2.json 

# ..... wait a bit...
update cattle completed
```
Now we can check:

```shell
$ build/infrakit group inspect cattle
ID                              LOGICAL         TAGS
instance-1475105646           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105656           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105666           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105676           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105686           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105696           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105706           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105716           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105726           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
instance-1475105736           	  -             infrakit.config_sha=BXedrwY0GdZlHhgHmPAzxTN4oHM=,infrakit.group=cattle,project=infrakit,tier=web
```

Note the instances now have a new SHA `BXedrwY0GdZlHhgHmPAzxTN4oHM=` (vs `Y23cKqyRpkQ_M60vIq7CufFmQWk=` previously)

To see that the Group plugin can enforce the size of the group, let's simulate an instance disappearing.

```shell
$ rm tutorial/instance-1475105646 tutorial/instance-1475105686 tutorial/instance-1475105726

# ... now check

$ ls -al tutorial
total 104
drwxr-xr-x  15 davidchung  staff   510 Sep 28 16:40 .
drwxr-xr-x  36 davidchung  staff  1224 Sep 28 16:39 ..
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:34 instance-1475105656
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:34 instance-1475105666
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:34 instance-1475105676
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:34 instance-1475105696
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:35 instance-1475105706
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:35 instance-1475105716
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:35 instance-1475105736
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:40 instance-1475106016 <-- new instance
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:40 instance-1475106026 <-- new instance
-rw-r--r--   1 davidchung  staff   654 Sep 28 16:40 instance-1475106036 <-- new instance
```

We see that 3 new instance have been created to replace the three removed, to match our
original specification of 10 instances.

Finally, let's clean up:

```shell
$ build/infrakit group destroy cattle
```

This concludes our quick tutorial.  In this tutorial we:
  + Started the plugins and learned to access them
  + Created a configuration for a group we wanted to watch
  + Verified the instances created matched the specifications
  + Updated the configurations of the group and scaled up the group
  + Reviewed the proposed changes
  + Applied the update across the group
  + Removed some instances and observed that the group self-healed
  + Destroyed the group
