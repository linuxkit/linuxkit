## Shell scripts
Shell scripts should loosely follow the general Alpine style which is derived from the Linux Kernel guidelines, i.e. tabs for indentation etc.

It's also useful to run `shellcheck` on the scripts.

## Go code
New Go code should be formatted with `gofmt`

## C code
C code written from scratch should follow the
[Linux kernel coding guidelines](https://git.kernel.org/cgit/linux/kernel/git/stable/linux-stable.git/tree/Documentation/CodingStyle)
as much as it makes sense for userspace code.  You can check your code with [checkpatch.pl](https://git.kernel.org/cgit/linux/kernel/git/stable/linux-stable.git/tree/scripts/checkpatch.pl) like this:
```
checkpatch.pl --no-tree --file <sourcefile>
```
