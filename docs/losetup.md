# LinuxKit losetup

Image to setup a loop device backed by a regular file in a [linuxkit](https://github.com/linuxkit/linuxkit)-generated image. The typical use case is to have a portable storage location which can be used to persist settings or other files. Can be combined with the `linuxkit/dm-crypt` package for protection.

## Usage

The setup is a one time step during boot:

```yaml
onboot:
  - name: losetup
    image: linuxkit/losetup:<hash>
    command: ["/usr/bin/loopy", "-c", "/var/test.img"]
```

The above will associate the file `/var/test.img` with `/dev/loop0` and will also create it if it's not present.

The container by default bind-mounts `/var:/var` and `/dev:/dev`. Usually the loop-file will reside on external storage which should be typically mounted under `/var` hence the choice of the defaults. If the loop-file is located somewhere else and you need a different bind-mount for it then do not forget to explicitly bind-mount `/dev:/dev` as well or else `losetup` will fail.

### Options

|Option|Default|Required|Notes|
|---|---|---|---|
|`-c` or `--create`||No|Creates the file if not present. If `--create` is not specified and the file is missing then the loop setup will obviously fail.|
|`-s` or `--size`|10|No|If `--create` was specified and the file is not present then this sets the size in MiB of the created file. The file will be filled from `/dev/zero`.|
|`-d` or `--dev`|`/dev/loop0`|No|Loop device which should be associated with the file.|
|`<file>`||**Yes**|The file to use as backing storage.|
