InfraKit
========

[![Circle 
CI](https://circleci.com/gh/docker/infrakit.png?style=shield&circle-token=50d2063f283f98b7d94746416c979af3102275b5)](https://circleci.com/gh/docker/infrakit)
[![codecov.io](https://codecov.io/github/docker/infrakit/coverage.svg?branch=master&token=z08ZKeIJfA)](https://codecov.io/github/docker/infrakit?branch=master)

_InfraKit_ is a toolkit for creating and managing declarative, self-healing infrastructure.
It breaks infrastructure automation down into simple, pluggable components. These components work together to actively
ensure the infrastructure state matches the user's specifications.
Although _InfraKit_ emphasizes primitives for building self-healing infrastructure, it also can be used passively like conventional tools.

![arch image](images/arch.png)

To get started, try the [tutorial](docs/tutorial.md).

## Plugins
_InfraKit_ leverages active processes, called _Plugins_, which can be composed to meet
different needs.  Technically, a Plugin is an HTTP server with a well-defined API, listening on a unix socket.

[Utilities](spi/http) are provided as libraries to simplify Plugin development in Go.

### Plugin types
#### Group
When managing infrastructure like computing clusters, Groups make good abstraction, and working with groups is easier
than managing individual instances. For example, a group can be made up of a collection
of machines as individual instances. The machines in a group can have identical configurations (replicas, or cattle).
They can also have slightly different properties like identity and ordering (as members of a quorum or pets).

_InfraKit_ provides primitives to manage Groups: a group has a given size and can shrink or grow based on some specification,
whether it's human generated or machine computed.
Group members can also be updated in a rolling fashion so that the configuration of the instance members reflect a new desired
state.  Operators can focus on Groups while _InfraKit_ handles the necessary coordination of Instances.

Since _InfraKit_ emphasizes on declarative infrastructure, there are no operations to move machines or Groups from one
state to another.  Instead, you _declare_ your desired state of the infrastructure.  _InfraKit_ is responsible
for converging towards, and maintaining, that desired state.

Therefore, a [group plugin](spi/group/spi.go) manages Groups of Instances and exposes the operations that are of interest to
a user:

  + watch/ unwatch a group (start / stop managing a group)
  + inspect a group
  + trigger an update the configuration of a group - like changing its size or underlying properties of instances. 
  + stop an update
  + destroy a group

##### Default Group plugin
_InfraKit_ provides a default Group plugin implementation, intended to suit common use cases.  The default Group plugin
manages Instances of a specific Flavor.  Instance and Flavor plugins can be composed to manage different types of
services on different infrastructure providers.

While it's generally simplest to use the default Group plugin, custom implementations may be valuable to adapt another
infrastructure management system.  This would allow you to use _InfraKit_ tooling to perform basic operations on widely
different infrastructure using the same interface.

#### Instance
Instances are members of a group. An [instance plugin](spi/instance/spi.go) manages some physical resource instances.
It knows only about individual instances and nothing about Groups.  Instance is technically defined by the plugin, and
need not be a physical machine at all.

For compute, for example, instances can be VM instances of identical spec. Instances
support the notions of attachment to auxiliary resources.  Instances are taggable and tags are assumed to be persistent
which allows the state of the cluster to be inferred and computed.

In some cases, instances can be identical, while in other cases the members of a group require stronger identities and
persistent, stable state. These properties are captured via the _flavors_ of the instances.

#### Flavor
Flavors help distinguish members of one group from another by describing how these members should be treated.
A [flavor plugin](spi/flavor/spi.go) can be thought of as defining what runs on an Instance.
It is responsible for dictating commands to run services, and check the health of those services.

Flavors allow a group of instances to have different characteristics.  In a group of cattle,
all members are treated identically and individual members do not have strong identity.  In a group of pets,
however, the members may require special handling and demand stronger notions of identity and state.


#### Reference implementations
This repository contains several Plugins which should be considered reference implementations for demonstration purposes
and development aides.  With the exception of those listed as
[supported](#supported-implementations), Plugins in this repository should be considered **not** to be under active
development and for use at your own risk.

Over time, we would prefer to phase out reference Plugins that appear to provide real value for implementations that
are developed independently.  For this reason, please [file an issue](https://github.com/docker/infrakit/issues/new)
to start a discussion before contributing to these plugins with non-trivial code.

| plugin                                            | type     | description                             |
|:--------------------------------------------------|:---------|:----------------------------------------|
| [swarm](plugin/flavor/swarm)                      | flavor   | runs Docker in Swarm mode               |
| [vanilla](plugin/flavor/vanilla)                  | flavor   | manual specification of instance fields |
| [zookeeper](plugin/flavor/zookeeper)              | flavor   | run an Apache ZooKeeper ensemble        |
| [infrakit/file](./example/instance/file)          | instance | useful for development and testing      |
| [infrakit/terraform](./example/instance/terraform)| instance | creates instances using Terraform       |
| [infrakit/vagrant](./example/instance/vagrant)    | instance | creates Vagrant VMs                     |


#### Supported implementations
The following Plugins are supported for active development.  Note that these Plugins may not be part of the InfraKit
project, so please double-check where the code lives before filing InfraKit issues.

| plugin                                                        | type     | description                                           |
|:--------------------------------------------------------------|:---------|:------------------------------------------------------|
| [infrakit/group](./cmd/group)                                 | group    | supports Instance and Flavor plugins, rolling updates |
| [docker/infrakit.aws](https://github.com/docker/infrakit.aws) | instance | creates Amazon EC2 instances                          |

Have a Plugin you'd like to share?  Submit a Pull Request to add yourself to the list!


## Building

### Your Environment

Make sure you check out the project following a convention for building Go projects. For example,

```shell

# Install Go - https://golang.org/dl/
# Assuming your go compiler is in /usr/local/go
export PATH=/usr/local/go/bin:$PATH

# Your dev environment
mkdir -p ~/go
export GOPATH=!$
export PATH=$GOPATH/bin:$PATH

mkdir -p ~/go/src/github.com/docker
cd !$
git clone git@github.com:docker/infrakit.git
cd infrakit
```

We recommended go version 1.7.1 or greater for all platforms.

Also install a few build tools

```shell
go get -u github.com/golang/lint/golint  # if you're running tests
```

### Running tests
```shell
$ make ci
```

### Binaries
```shell
$ make binaries
```
Executables will be placed in the `./build` directory.
This will produce binaries for tools and several reference Plugin implementations:
  + [`infrakit`](./cmd/cli/README.md): a command line interface to interact with plugins
  + [`infrakit-group-default`](./cmd/group/README.md): the default [Group plugin](./spi/group)
  + [`infrakit-instance-file`](./example/instance/file): an Instance plugin using dummy files to represent instances
  + [`infrakit-instance-terraform`](./example/instance/terraform):
    an Instance plugin integrating [Terraform](https://www.terraform.io)
  + [`infrakit-instance-vagrant`](./example/instance/vagrant):
    an Instance plugin using [Vagrant](https://www.vagrantup.com/)
  + [`infrakit-flavor-vanilla`](./example/flavor/vanilla):
    a Flavor plugin for plain vanilla set up with user data and labels
  + [`infrakit-flavor-zookeeper`](./example/flavor/zookeeper):
    a Flavor plugin for [Apache ZooKeeper](https://zookeeper.apache.org/) ensemble members
  + [`infrakit-flavor-swarm`](./example/flavor/swarm):
    a Flavor plugin for Docker in [Swarm mode](https://docs.docker.com/engine/swarm/).

All provided binaries have a `help` subcommand to get usage and a `version` subcommand to identify the build revision.

## Examples
There are a few examples of _InfraKit_ plugins:

  + Terraform Instance Plugin
    - [README](./example/instance/terraform/README.md)
    - [Code] (./example/instance/terraform/plugin.go) and [configs](./example/instance/terraform/aws-two-tier)
  + Zookeeper / Vagrant
    - [README](./example/flavor/zookeeper/README.md)
    - [Code] (./plugin/flavor/zookeeper)


# Design

## Configuration
_InfraKit_ uses JSON for configuration because it is composable and a widely accepted format for many
infrastructure SDKs and tools.  Since the system is highly component-driven, our JSON format follows
simple patterns to support the composition of components.

A common pattern for a JSON object looks like this:

```json
{
   "SomeKey": "ValueForTheKey",
   "Properties": {
   }
}
```

There is only one `Properties` field in this JSON and its value is a JSON object. The opaque
JSON value for `Properties` is decoded via the Go `Spec` struct defined within the package of the plugin --
for example -- [`vanilla.Spec`](/plugin/flavor/vanilla/flavor.go).

The JSON above is a _value_, but the type of the value belongs outside the structure.  For example, the
default Group [Spec](/plugin/group/types/types.go) is composed of an Instance plugin, a Flavor plugin, and an
Allocation:

```json
{
  "ID": "name-of-the-group",
  "Properties": {
    "Allocation": {
    },
    "Instance": {
      "Plugin": "name-of-the-instance-plugin",
      "Properties": {
      }
    },
    "Flavor": {
      "Plugin": "name-of-the-flavor-plugin",
      "Properties": {
      }
    }
  }
}
```
The group's Spec has `Instance` and `Flavor` fields which are used to indicate the type, and the value of the
fields follow the pattern of `<some_key>` and `Properties` as shown above.

The `Allocation` determines how the Group is managed.  Allocation has two properties:
  - `Size`: an integer for the number of instances to maintain in the Group
  - `LogicalIDs`: a list of string identifiers, one will be associated wih each Instance

Exactly one of these fields must be set, which defines whether the Group is treated as 'cattle' (`Size`) or 'pets'
(`LogicalIDs`).  It is up to the Instance and Flavor plugins to determine how to use `LogicalID` values.

As an example, if you wanted to manage a Group of NGINX servers, you could
write a custom Group plugin for ultimate customizability.  The most concise configuration looks something like this:

```json
{
  "ID": "nginx",
  "Plugin": "my-nginx-group-plugin",
  "Properties": {
    "port": 8080
  }
}
````

However, you would likely prefer to use the default Group plugin and implement a Flavor plugin to focus on
application-specific behavior.  This gives you immediate support for any infrastructure that has an Instance plugin.
Your resulting configuration might look something like this:

```json
{
  "ID": "nginx",
  "Plugin": "group",
  "Properties": {
    "Allocation": {
      "Size": 10
    },
    "Instance": {
      "Plugin": "aws",
      "Properties": {
        "region": "us-west-2",
        "ami": "ami-123456"
      }
    },
    "Flavor": {
      "Plugin": "nginx",
      "Properties": {
        "port": 8080
      }
    }
  }
}
```

Once the configuration is ready, you will tell a Group plugin to
  + watch it
  + update it
  + destroy it

Watching the group as specified in the configuration means that the Group plugin will create
the instances if they don't already exist.  New instances will be created if for any reason
existing instances have disappeared such that the state doesn't match your specifications.

Updating the group tells the Group plugin that your configuration may have changed.  It will
then determine the changes necessary to ensure the state of the infrastructure matches the new
specification.

## Plugin Discovery

Multiple _InfraKit_ plugins are typically used together to support a declared configuration.  These plugins discover
each other by looking for socket files in a common plugin directory, and communicate via HTTP.

The default plugin directory is `~/.infrakit/plugins`, and can be overridden with the environment variable
`INFRAKIT_PLUGINS_DIR`.

Note that multiple instances of a plugin may run, provided they have different names for discovery.  This may be useful,
for example, if a plugin can be configured to behave differently. For example:

The CLI shows which plugins are [discoverable](cmd/cli/README.md#list-plugins).

## Docs

Design docs can be found [here](./docs).

## Reporting security issues

The maintainers take security seriously. If you discover a security issue,
please bring it to their attention right away!

Please **DO NOT** file a public issue, instead send your report privately to
[security@docker.com](mailto:security@docker.com).

Security reports are greatly appreciated and we will publicly thank you for it.
We also like to send gifts—if you're into Docker schwag, make sure to let
us know. We currently do not offer a paid security bounty program, but are not
ruling it out in the future.


## Copyright and license

Copyright © 2016 Docker, Inc. All rights reserved. Released under the Apache 2.0
license. See [LICENSE](LICENSE) for the full license text.
