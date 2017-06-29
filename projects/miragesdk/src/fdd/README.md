## fdd -- file-descriptor daemon

`Fdd` allows to share socketpair over a unix domain socket. The typical flow is
as follows:

1. Start the fdd daemon:
    ```
    $ fdd init
   ```

2. Create a bunch of socketpair shares:
    ```
    $ fdd share /tmp/foo
    $ fdd share /tmp/bar
    ```

This will create `/tmp/foo` and `/tmp/bar` that process clients can connect too.
Once connected, they can use `recvmsg`[1] to receive each side of the
socketpair. If two different process do this, they then have a channel to talk
to each other.

[1]: https://linux.die.net/man/2/recvmsg
