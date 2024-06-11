package moby

var schema = `
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Moby Config",
  "additionalProperties": false,
  "definitions": {
    "kernel": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "image": {"type": "string"},
        "cmdline": {"type": "string"},
        "binary": {"type": "string"},
        "tar": {"type": "string"},
        "ucode": {"type": "string"}
      }
    },
    "file": {
      "type": "object",
      "additionalProperties": false,
        "properties": {
          "path": {"type": "string"},
          "directory": {"type": "boolean"},
          "symlink": {"type": "string"},
          "contents": {"type": "string"},
          "source": {"type": "string"},
          "metadata": {"type": "string"},
          "optional": {"type": "boolean"},
          "mode": {"type": "string"},
          "uid": {"anyOf": [{"type": "string"}, {"type": "integer"}]},
          "gid": {"anyOf": [{"type": "string"}, {"type": "integer"}]}
        }
    },
    "files": {
        "type": "array",
        "items": { "$ref": "#/definitions/file" }
    },
    "trust": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "image": { "$ref": "#/definitions/strings" },
        "org": { "$ref": "#/definitions/strings" }
      }
    },
    "strings": {
        "type": "array",
        "items": {"type": "string"}
    },
    "mapstring": {
        "type": "object",
        "additionalProperties": {"type": "string"}
    },
    "mount": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "destination": { "type": "string" },
        "type": { "type": "string" },
        "source": { "type": "string" },
        "options": { "$ref": "#/definitions/strings" },
        "uidmappings": {
          "type": "array",
          "items": { "$ref": "#/definitions/idmapping" }
        },
        "gidmappings": {
          "type": "array",
          "items": { "$ref": "#/definitions/idmapping" }
        }
      }
    },
    "mounts": {
      "type": "array",
      "items": { "$ref": "#/definitions/mount" }
    },
    "device": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "path": { "type": "string" },
        "type": { "type": "string" },
        "major": { "type": "integer" },
        "minor": { "type": "integer" },
        "mode": { "type": "string" }
      }
    },
    "devices": {
      "type": "array",
      "items": { "$ref": "#/definitions/device" }
    },
    "idmapping": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "hostID": { "type": "integer" },
        "containerID": { "type": "integer" },
        "size": { "type": "integer" }
      }
    },
    "idmappings": {
      "type": "array",
      "items": { "$ref": "#/definitions/idmapping" }
    },
    "devicecgroups": {
      "type": "array",
      "items": { "$ref": "#/definitions/devicecgroup" }
    },
    "devicecgroup": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "allow": {"type": "boolean"},
        "type": {"type": "string"},
        "major": {"type": "integer"},
        "minor": {"type": "integer"},
        "access": {"type": "string"}
      }
    },
    "memory": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "limit": {"type": "integer"},
        "reservation": {"type": "integer"},
        "swap": {"type": "integer"},
        "kernel": {"type": "integer"},
        "kernelTCP": {"type": "integer"},
        "swappiness": {"type": "integer"},
        "disableOOMKiller": {"type": "boolean"}
      }
    },
    "cpu": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "shares": {"type": "integer"},
        "quota": {"type": "integer"},
        "period": {"type": "integer"},
        "realtimeRuntime": {"type": "integer"},
        "realtimePeriod": {"type": "integer"},
        "cpus": {"type": "string"},
        "mems": {"type": "string"}
      }
    },
    "pids": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "limit": {"type": "integer"}
      }
    },
    "weightdevices": {
      "type": "array",
      "items": {"$ref": "#/definitions/weightdevice"}
    },
    "weightdevice": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "major": {"type": "integer"},
        "minor": {"type": "integer"},
        "weight": {"type": "integer"},
        "leafWeight": {"type": "integer"}
      }
    },
    "throttledevices": {
      "type": "array",
      "items": {"$ref": "#/definitions/throttledevice"}
    },
    "throttledevice": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "major": {"type": "integer"},
        "minor": {"type": "integer"},
        "rate": {"type": "integer"}
      }
    },
    "blockio": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "weight": {"type": "integer"},
        "leafWeight": {"type": "integer"},
        "weightDevice": {"$ref": "#/definitions/weightdevices"},
        "throttleReadBpsDevice": {"$ref": "#/definitions/throttledevices"},
        "throttleWriteBpsDevice": {"$ref": "#/definitions/throttledevices"},
        "throttleReadIOPSDevice": {"$ref": "#/definitions/throttledevices"},
        "throttleWriteIOPSDevice": {"$ref": "#/definitions/throttledevices"}
      }
    },
    "hugepagelimits": {
      "type": "array",
      "items": {"$ref": "#/definitions/hugepagelimit"}
    },
    "hugepagelimit": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "pageSize": {"type": "integer"},
        "limit": {"type": "integer"}
      }
    },
    "interfacepriorities": {
      "type": "array",
      "items": {"$ref": "#/definitions/interfacepriority"}
    },
    "interfacepriority": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "name": {"type": "string"},
        "priority": {"type": "integer"}
      }
    },
    "network": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "classID": {"type": "integer"},
        "priorities": {"$ref": "#/definitions/interfacepriorities"}
      }
    },
    "resources": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "devices": {"$ref": "#/definitions/devicecgroups"},
        "memory": {"$ref": "#/definitions/memory"},
        "cpu": {"$ref": "#/definitions/cpu"},
        "pids": {"$ref": "#/definitions/pids"},
        "blockio": {"$ref": "#/definitions/blockio"},
        "hugepageLimits": {"$ref": "#/definitions/hugepagelimits"},
        "network": {"$ref": "#/definitions/network"}
      }
    },
    "interfaces": {
      "type": "array",
      "items": {"$ref": "#/definitions/interface"}
    },
    "interface": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "name": {"type": "string"},
        "add": {"type": "string"},
        "peer": {"type": "string"},
        "createInRoot": {"type": "boolean"}
      }
    },
    "namespaces": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "cgroup": {"type": "string"},
        "ipc": {"type": "string"},
        "mnt": {"type": "string"},
        "net": {"type": "string"},
        "pid": {"type": "string"},
        "user": {"type": "string"},
        "uts": {"type": "string"}
      }
    },
    "runtime": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "cgroups": {"$ref": "#/definitions/strings"},
        "mounts": {"$ref": "#/definitions/mounts"},
        "mkdir": {"$ref": "#/definitions/strings"},
        "interfaces": {"$ref": "#/definitions/interfaces"},
        "bindNS": {"$ref": "#/definitions/namespaces"},
        "namespace": {"type": "string"}
      }
    },
    "image": {
      "type": "object",
      "additionalProperties": false,
      "required": ["name", "image"],
      "properties": {
        "name": {"type": "string"},
        "image": {"type": "string"},
        "capabilities": { "$ref": "#/definitions/strings" },
        "capabilities.add": { "$ref": "#/definitions/strings" },
        "ambient": { "$ref": "#/definitions/strings" },
        "mounts": { "$ref": "#/definitions/mounts" },
        "binds": { "$ref": "#/definitions/strings" },
        "binds.add": { "$ref": "#/definitions/strings" },
        "devices": { "$ref": "#/definitions/devices" },
        "tmpfs": { "$ref": "#/definitions/strings" },
        "command": { "$ref": "#/definitions/strings" },
        "env": { "$ref": "#/definitions/strings" },
        "cwd": { "type": "string"},
        "net": { "type": "string"},
        "pid": { "type": "string"},
        "ipc": { "type": "string"},
        "uts": { "type": "string"},
        "userns": { "type": "string"},
        "readonly": { "type": "boolean"},
        "maskedPaths": { "$ref": "#/definitions/strings" },
        "readonlyPaths": { "$ref": "#/definitions/strings" },
        "uid": {"anyOf": [{"type": "string"}, {"type": "integer"}]},
        "gid": {"anyOf": [{"type": "string"}, {"type": "integer"}]},
        "additionalGids": {
            "type": "array",
            "items": {"anyOf": [{"type": "string"}, {"type": "integer"}]}
        },
        "noNewPrivileges": {"type": "boolean"},
        "hostname": {"type": "string"},
        "oomScoreAdj": {"type": "integer"},
        "rootfsPropagation": {"type": "string"},
        "cgroupsPath": {"type": "string"},
        "resources": {"$ref": "#/definitions/resources"},
        "sysctl": { "$ref": "#/definitions/mapstring" },
        "rlimits": { "$ref": "#/definitions/strings" },
        "uidMappings": { "$ref": "#/definitions/idmappings" },
        "gidMappings": { "$ref": "#/definitions/idmappings" },
        "annotations": { "$ref": "#/definitions/mapstring" },
        "runtime": {"$ref": "#/definitions/runtime"}
      }
    },
    "images": {
        "type": "array",
        "items": { "$ref": "#/definitions/image" }
    }
  },
  "properties": {
    "kernel": { "$ref": "#/definitions/kernel" },
    "init": { "$ref": "#/definitions/strings" },
    "onboot": { "$ref": "#/definitions/images" },
    "onshutdown": { "$ref": "#/definitions/images" },
    "services": { "$ref": "#/definitions/images" },
    "trust": { "$ref": "#/definitions/trust" },
    "files": { "$ref": "#/definitions/files" }
  }
}
`
