## kernel config project

The intent of the kernel config project is to demonstrate a better way to
handle kernel config. Specifically:

* support for arch and version specific config
* make diffs as readable as possible
* ensure that all of our config settings are kept after oldconfig

We achieve the goals by:

* having version-specific config in separate files, which are automatically
  merged
* only keeping track of visible symbols, only keeping track of a delta from
  defconfig, and keeping symbols sorted alphabetically
* checking after a `make oldconfig` in the kernel, that all of our symbols are
  set as we want them to be

The bulk of this work happens in makeconfig.sh, which merges the configs (and
checks that the resulting config is okay).

One important piece is generating a kernel config for a new version. There are
a few cases:

* A new kconfig symbol is introduced that we want to set a non-default value
  of: in this case, we introduce a new `kernel_config.${VERSION}` file, and set
  the value to what we want to set it to
* A config symbol that was no-default before become the default: in this case,
  we would move the non-default setting to version specific files for all of
  the other versions, and not set anything for this new kernel, since what we
  want is now the default.
* A symbol we want to set is removed (or renamed), similar to the above, we
  simply move the old symbol name to version specific files for older kernels
  and put the new symbol name (if it exists) in the new version specific file

When dropping support for an old kernel version, we just delete that version
specific file, and promote any option that is present in all other versions to
the common config file.
