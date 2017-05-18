## IMA

IMA stands for Integrity Management Architecture. The basic idea is to prevent
userspace from even *opening* files that have been mutated, by tracking file
content via a hash in the `security.ima` extended attribute. IMA supports
keeping track of these hashes and signing the result via the TPM, and a host of
other features.

Today, this is not namespace aware, so there is no way to differentiate in
IMA's appraisal output between files in one mount namespace vs another, which
makes this not particularly useful for container engines. The goal of this
patchset is to make IMA namespace aware.

## IMA namespace patches

These are draft patches for an implementation of IMA namespacing. They are
currently a rebased version of the v1 set posted here [1].

### Usage

Let's suppose you have some sensitive files owned by a particular user that you
want to keep secure:

    sensitive=/tmp/foo
    user=71452
    mkdir -p $(dirname $sensitive) && echo "hello" > $sensitive
    chown $user $sensitive

To use IMA in the per-namespace mode, you need ima\_appraise=enforce\_ns on the
kernel CLI (this is done in the yaml file). Then, the userspace interface looks
something like this:

    # create a new mount namespace
    unshare -m

    # enable per-ns policy for this new namespace
    nsid=$(readlink /proc/self/ns/mnt | cut -c '6-15')
    echo ${nsid} > /sys/kernel/security/ima/namespaces

    # set the policy (we use tmpfs magic here since that's all that linuxkit
    # has available to write to for this example)
    TMPFS_MAGIC=0x01021994
    printf "appraise fsmagic=$TMPFS_MAGIC fowner=$user\nappraise func=MODULE_CHECK" > /sys/kernel/security/ima/$nsid/policy

    hash=$(echo -e "\x4$(openssl dgst -sha256 -binary $sensitive)")
    setfattr -n security.ima -v "${hash}" $sensitive

And now you should be able to see things failing:

    moby:/# echo foo > /tmp/foo
    moby:/# cat /tmp/foo 
    [ 3233.681544] audit: type=1800 audit(1495131746.610:29): pid=384 uid=0 auid=4294967295 ses=4294967295 op="appraise_data" cause="invalid-hash" comm="cat" name="/tmp/foo" mnt_ns=4026532208 dev="tmpfs" ino=13105 res=0
    cat: can't open '/tmp/foo': Permission denied

[1]: https://lkml.org/lkml/2017/5/11/699
