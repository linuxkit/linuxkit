# LinuxKit getty
Image to create a getty on each console for a [linuxkit](https://github.com/linuxkit/linuxkit)-generated image.

## Usage
LinuxKit images do not create `getty` by default. If you want to be able to access a shell, you need to run a getty.

If you want a console getty, add the following to your `moby.yml`:

```
services:
  - name: getty
    image: linuxkit/getty:<hash>
```

The above will launch a getty for each console defined in the cmdline, i.e. `/proc/cmdline`.


### securetty
Every console defined in the `cmdline` **must** also already exist in `/etc/securetty` if you wish to login on that tty as root. If it does not exist, a getty will be started, but you will not be able to login as root. A warning message will be sent to that tty.

If you are using a console that is not in `securetty`, you can add it by overriding the default `securetty` file in the linuxkit root filesystem using `files:` in your moby `.yml` file.


### Login Options
There are 3 ways to launch a getty on a linuxkit instance:

1. Login disabled
2. Password login
3. Open access


#### Login Disabled
Login disabled prevents any console login. This is the most secure option and recommended for production deployments.

To disable login entirely:

1. Ensure you are running a version of `linuxkit/init` that has getty disabled.
2. Do **not** add `linuxkit/getty` as a `service`

Conversely, you can include `linuxkit/getty` as a `service`, but do not map in an `/etc/shadow` file. Since the default root password is blocked, this, too, will prevent login. However, we strongly recommend simply not enabling `linuxkit/getty` if you desire to block login.


#### Password Login
Password login is like traditional login. At the console, you get a prompt, and enter your username and password.

To enable password login, you must provide getty with the root password. You do so by creating a file `/etc/getty.shadow` in the linuxkit host. For example:

```yml
files:
  - path: etc/getty.shadow
    # sample sets password for root to "abcdefgh" (without quotes)
    contents: 'root:$6$6tPd2uhHrecCEKug$8mKfcgfwguP7f.BLdZsT1Wz7WIIJOBY1oUFHzIv9/O71M2J0EPdtFqFGTxB1UK5ejqQxRFQ.ZSG9YXR0SNsc11:17322:0:::::'
```

Note that `/etc/shadow` is sensitive to having a carriage return at the end of each line. To be safe, the `getty` container will add a newline at the end of a mapped shadow file.

The `linuxkit/getty` container already is set up to map `/etc/getty.shadow` to `/etc/shadow`.

The existing `/etc/password` has a single line with `root` as UID `0`; your `/etc/shadow` should match that.

If no `/etc/shadow` os provided, the login will be unusable, as the default `root` user has a blocked password.

#### Open Access
With open access, no password is required. Any user accessing the console will immediately get a root login shell.

To enable open access, you must tell getty explicitly that you wish to have insecure access by setting the environment variable `INSECURE=true` for the container.

## Example
An example yml file is included in [examples/getty.yml](../../examples/getty.yml). The sample uses a custom root password, and comments describing how to make it insecure instead.


## LinuxKit Debug
In addition to the usual getty shell, it is possible that you have a LinuxKit build that is failing, to the point where even `containerd` is not starting correctly, or not launching services. In such a case, `getty` will not run, since `containerd` launches it. This leaves you with no ability to log onto the system and debug it.

In that case, you can make `linuxkit/getty` an `init:` level container. This will lead to a `sh` running on the console.

**This is highly insecure and should not be used except to debug system startup where containerd will not start itself or services. In all other cases, use getty only via services.**
