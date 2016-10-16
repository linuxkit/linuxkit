InfraKit CLI
============

This is a developer CLI for working with various _InfraKit_ plugins.  The CLI offers several subcommands for working
with plugins. In general, plugin methods are exposed as verbs and configuration JSON can be read from local file.

## Building

Begin by building plugin [binaries](../../README.md#binaries).

### List Plugins

```
$ build/infrakit plugin ls
Plugins:
NAME                	LISTEN
flavor-swarm        	~/.infrakit/plugins/flavor-swarm
flavor-zookeeper    	~/.infrakit/plugins/flavor-zookeeper
group               	~/.infrakit/plugins/group
instance-file       	~/.infrakit/plugins/instance-file
```

Once you know the plugins by name, you can make calls to them.  For example, the instance plugin
`instance-file` is a Plugin that "provisions" instances by writing the instructions to
a file in a local directory.

You can access the following plugins and their methods via command line:

  + instance
  + group
  + flavor

### Working with Instance Plugin

Using the plugin `instance-file` as an example:

### Validate

```
$ cat << EOF > instance.json
{
    "Properties": {
        "version": "v0.0.1"
    },
    "Tags": {
        "instanceType": "small",
        "group": "test2"
    },
    "Init": "#!/bin/sh\napt-get install -y wget",
    "LogicalID": "logic2"
}
EOF

$ build/infrakit instance --name instance-file instance.json
validate:ok
```

### Provision

```
$ build/infrakit instance --name instance-file provision instance.json
instance-1474873473
```

### List instances

```
$ build/infrakit instance --name instance-file describe
ID                            	LOGICAL                       	TAGS
instance-1474850397           	  -                           	group=test,instanceType=small
instance-1474850412           	  -                           	group=test2,instanceType=small
instance-1474851747           	logic2                        	group=test2,instanceType=small
instance-1474873473           	logic2                        	group=test2,instanceType=small
```

### Destroy

```
$ build/infrakit instance --name instance-file destroy instance-1474873473
destroyed instance-1474873473
```
