# Device encryption with dm-crypt

In the packages section you can find an image to setup dm-crypt encrypted devices in [linuxkit](https://github.com/linuxkit/linuxkit)-generated images.

## Usage

The setup is a one time step during boot:

```yaml
onboot:
  - name: dm-crypt
    image: linuxkit/dm-crypt:<hash>
    command: ["/usr/bin/crypto", "dm_crypt_name", "/dev/sda1"]
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "/dev/mapper/dm_crypt_name", "/var/secure_storage"]
files:
  - path: etc/dm-crypt/key
    contents: "abcdefghijklmnopqrstuvwxyz123456"
```

The above will map `/dev/sda1` as an encrypted device under `/dev/mapper/dm_crypt_name` and mount it under `/var/secure_storage`

The `dm-crypt` container by default bind-mounts `/dev:/dev` and `/etc/dm-crypt:/etc/dm-crypt`. It expects the encryption key to be present in the file `/etc/dm-crypt/key`. You can pass an alternative location as encryption key which can be either a file path relative to `/etc/dm-crypt` or an absolute path.

Providing an alternative encryption key file name:

```yaml
onboot:
  - name: dm-crypt
    image: linuxkit/dm-crypt:<hash>
    command: ["/usr/bin/crypto", "-k", "some_other_key", "dm_crypt_name", "/dev/sda1"]
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "/dev/mapper/dm_crypt_name", "/var/secure_storage"]
files:
  - path: etc/dm-crypt/some_other_key
    contents: "abcdefghijklmnopqrstuvwxyz123456"
```

Providing an alternative encryption key file name as absolute path:

```yaml
onboot:
  - name: dm-crypt
    image: linuxkit/dm-crypt:<hash>
    command: ["/usr/bin/crypto", "-k", "/some/other/key", "dm_crypt_name", "/dev/sda1"]
    binds:
      - /dev:/dev
      - /etc/dm-crypt/some_other_key:/some/other/key
  - name: mount
    image: linuxkit/mount:<hash>
    command: ["/usr/bin/mountie", "/dev/mapper/dm_crypt_name", "/var/secure_storage"]
files:
  - path: etc/dm-crypt/some_other_key
    contents: "abcdefghijklmnopqrstuvwxyz123456"
```

Note that you have to also map `/dev:/dev` explicitly if you override the default bind-mounts.

The `dm-crypt` container

* Will create an `ext4` file system on the encrypted device if none is present.
  * It will also initialize the encrypted device by filling it from `/dev/zero` prior to creating the filesystem. Which means if the device is being setup for the first time it might take a bit longer.
* Uses the `aes-cbc-essiv:sha256` cipher (it's explicitly specified in case the default ever changes)
  * Consequently the encryption key is expected to be 32 bytes long, a random one can be created via
    ```shell
    dd if=/dev/urandom of=dm-crypt.key bs=32 count=1
    ```
    If you see the error `Cannot read requested amount of data.` next to the log message `Creating dm-crypt mapping for ...` then this means your keyfile doesn't contain enough data.

### Examples

There are two examples in the `examples/` folder:

1. `dm-crypt.yml` - formats an external disk and mounts it encrypted.
2. `dm-crypt-loop.yml` - mounts an encrypted loop device backed by a regular file sitting on an external disk

### Options

|Option|Default|Required|Notes|
|---|---|---|---|
|`-k` or `--key`|`key`|No|Encryption key file name. Must be either relative to `/etc/dm-crypt` or an absolute file path.|
|`-l` or `--luks`||No|Use LUKS format for encryption|
|`<dm_name>`||**Yes**|The device-mapper device name to use. The device will be mapped under `/dev/mapper/<dm_name>`|
|`<device>`||**Yes**|Device to encrypt.|
