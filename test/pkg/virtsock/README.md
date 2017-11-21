This directory contains the files to build and run a container containing the
virtio and Hyper-V socket stress tests. `../../cases/test-virtsock-server.yml` builds images which start the server inside the VM.

The client, to be run on the host as per this [README](https://github.com/rneugeba/virtsock/blob/master/examples/README.md), can be obtained compiled from [here](https://github.com/rneugeba/virtsock).

## How to use (on Windows)

- Build the images: `linuxkit build tests/cases/test-virtsock-server.yml`
- Copy the `test-virtsock-server.iso` to a Windows system
- Create a Type 1 Hyper-V VM (called `virtsock`).
  - No Disk or network required
  - Add the ISO to the CDROM device
  - Make sure you enable a named pipe for COM1 (call it `virtsock`)
- Start the VM
- Connect to the serial console (to get debug output) with `putty -serial \\.\pipe\virtsock`

Run the client:
```
$vmId = (get-vm virtsock).Id
.\virtsock_stress.exe -c $vmId  -v 1 -c 1000000 -p 10
```

This creates `1000000` connections from `10` threads to the VM and
sends some random amount of data of the connection before tearing it
down. There are more options to change the behaviour.


## TODO

- Add scripts to create Hyper-V VM
- Enable virtio sockets in `linuxkit run` with HyperKit
- Add some sample client YAML files which would connect from the VM to the host
- Hook up to CI for both HyperKit and Hyper-V
