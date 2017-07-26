# LinuxKit tpm
Image to provide a tcsd daemon and tpm tools to support tpm, based on [trousers and tpm-tools](https://sourceforge.net/projects/trousers/) .


## Usage
If you want to interact with a tpm chip, you need a `tcsd` daemon running to control it and provide a communication endpoint for all of your tpm commands.

This image provides both a `tcsd` daemon to run in a container, and the command line `tpm-tools`.

### Daemon
To run a `tcsd` daemon - you **must** run exactly one on a tpm-enabled host to interact with the tpm - just start the container.

#### LinuxKit
In LinuxKit, add the following to your moby `.yml`:

```
services:
  - name: tcsd
    image: "secureapp/tss:<hash>"
```

The above will launch `tcsd` listening on localhost only.

#### Docker
In regular docker or other container environment, start the container in the background. Be sure to map `/dev:/dev` and expose port `30003`.

```
docker run -d -v /dev:/dev -p 30003:30003 secureapp/tcsd
```
### CLI Tools
To run the CLI tools, just run them:

```
docker run -it --rm secureapp/tcsd tpm_nvread
```
