# LinuxKit Swap
Image to enable creation of a swap file for a [linuxkit](https://github.com/linuxkit/linuxkit)-generated image.


## Usage
Normally, unless you are running explicitly in a desktop version, LinuxKit images do not have swap enabled. If you want swap, add the following to your `moby.yml`:

```
onboot:
  - name: swap
    image: linuxkit/swap:<hash>
    command: ["/swap.sh","--path","/var/external/swap","--size","2G"]
```

Note that you **must** mount the following:

* `/var` to `/var` so it can place the swapfile in the right location.
* `/dev` to `/dev` so it can do the right thing for devices

### Options

Options are passed to it via command-line options. The following are the options. Details follow.

|Option|Parameter|Default|Required|Notes|
|---|---|---|---|---|
|`--path`|Path to file as seen in the underlying OS||**Yes**||
|`--size`|Target swapfile size||**Yes**||
|`--condition`|_condition_||No|Condition that must be met to create a swapfile|
|`--debug`|||No|Turns on verbose output from the command making the swap|
|`--encrypt`|||No|Encrypts swapfile|


#### File
You can create a swapfile at the given path. You **must** provide exactly one swapfile path, or none will be created; there is no default. Passing fewer than or more than one `--path` option causes the container to exit with an error.

The option `--path` takes a single argument _path_, path to the swapfile, e.g. `/var/mnt/swap2`.

You **always** should put the swap file somewhere under `/var`, since that is where read-writable files and mounts exist in linuxkit images.

#### Size
`--size <size>` indicates the desired swapfile size, e.g. `2G` `100M` `5670K` `8765432`. There is no default. Acceptable size units are `G`, `M`, `K` and bytes of no unit provided.

If disk space on the requested partition is insufficient to create the swapfile, the container exits with an error.

#### Encryption
If you want the swapfile to be encrypted, pass the `--encrypt` option. It will create an encrypted swapfile at the path you provide to `--path`, using devicemapper to map the clear device to `/dev/mapper/swapfile`.

Encryption is performed using `cryptsetup` with `plain` encryption, using `/dev/urandom` to generate a random keyfile, key size of `256`, and cipher `aes-cbc-essiv:sha256`.

#### Conditions
You may want to create a swapfile only if certain conditions are met. Supported conditions are:

* An external disk is available
* Partition on which the swapfile will sit is of a minimum size.

**All** conditions must be met. If a condition fails, the swapfile is not created, but the container exits with no error, unless you set the condition as `required`.

Conditions are structured as follows:    _type_:_parameter_:_required_

In all cases, you may leave off _required_ if you wish to treat it as false, i.e. do not create swapfile but exit with no error if the condition is not met.

##### Partition exists
LinuxKit may be running from a small disk, and you only want to run if a particular large external disk is available. In that case, pass `--condition part:<path>:<required>` to indicate that swapfile is to be created only if _path_ already exists and is a mount point.

Example: `--condition part:/var/mnt/external`

##### Size
You may set a minimum size for a partition (likely the one on which the swapfile will be created) using the `size` condition of the format `--condition size:<path>:<size>:<required>` to indicate that swapfile is to be created only if the partition on which _path_ exists is of minimum size _size_. Acceptable sizes are identical to those for the swapfile.

Examples:

* `--condition partsize:/var/mnt/external:100G:true`


## Example
An example yml file is included in [examples/swap.yml](../../examples/swap.yml). `swap.yml`. Note that you need to attach an external disk - trying to create a swapfile in tmpfs `/var` will fail.

The sample command to run the enclosed is:

```
linuxkit build swap.yml
linuxkit run -disk size=4G swap
```
