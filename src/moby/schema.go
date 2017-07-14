package moby

var schema = string(`
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
        "tar": {"type": "string"}
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
          "optional": {"type": "boolean"},
          "mode": {"type": "string"},
          "uid": {"type": "string"},
          "gid": {"type": "string"}
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
    "mount": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "destination": { "type": "string" },
        "type": { "type": "string" },
        "source": { "type": "string" },
        "options": { "$ref": "#/definitions/strings" }
      }
    },
    "mounts": {
      "type": "array",
      "items": { "$ref": "#/definitions/mount" }
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
    "image": {
      "type": "object",
      "additionalProperties": false,
      "required": ["name", "image"],
      "properties": {
        "name": {"type": "string"},
        "image": {"type": "string"},
        "capabilities": { "$ref": "#/definitions/strings" },
        "ambient": { "$ref": "#/definitions/strings" },
        "mounts": { "$ref": "#/definitions/mounts" },
        "binds": { "$ref": "#/definitions/strings" },
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
        "uid": {"type": "string"},
        "gid": {"type": "string"},
        "additionalGids": {
            "type": "array",
            "items": { "type": "string" }
        },
        "noNewPrivileges": {"type": "boolean"},
        "hostname": {"type": "string"},
        "oomScoreAdj": {"type": "integer"},
        "disableOOMKiller": {"type": "boolean"},
        "rootfsPropagation": {"type": "string"},
        "cgroupsPath": {"type": "string"},
        "sysctl": {
            "type": "array",
            "items": { "$ref": "#/definitions/strings" }
        },
        "rlimits": { "$ref": "#/definitions/strings" },
        "uidMappings": { "$ref": "#/definitions/idmappings" },
        "gidMappings": { "$ref": "#/definitions/idmappings" }
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
    "services": { "$ref": "#/definitions/images" },
    "trust": { "$ref": "#/definitions/trust" },
    "files": { "$ref": "#/definitions/files" }
  }
}
`)
