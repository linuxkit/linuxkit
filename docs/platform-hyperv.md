# LinuxKit with Hyper-V

LinuxKit supports running LinuxKit created VMs on Hyper-V on Microsoft
Windows. `linuxkit` must be run from an elevated command prompt,
typically a elevated Powershell and utilises Powershell scripting to
manage the Hyper-V VMs.


Example:
```sh
linuxkit.exe run --disk size=1 linuxkit-efi.iso
```

The Hyper-V VM, by default, is named after the prefix of the ISO, ie
with `.iso` and `-efi` stripped. Note, You may only have one VM for a
given name.  You can specify an alternative name using the `-name`
command-line option.


## Boot

Currently, the Hyper-V backend only supports booting EFI ISO images
created with LinuxKit. `linuxkit` create a Generation 2 VM and
disables secure EFI boot for booting.

In the future, we may add support for legacy BIOS ISOs, booting from
disks, and enable secure boot.


## Console

The serial port of the VM is configured to redirect to a Named Pipe,
and when the `linuxkit` command is executed an interactive console is
provided in the same window. The serial console may also be redirected
to a file.

**Note:** The connection to the Named Pipe sometimes seems to be a bit
racey, though the code itself should be fine. You may have to try
again if the connection to the serial console fails.

If the main console is configured within the VM, one can also connect
to it using the Hyper-V manager, or from the command-line:
```sh
vmconnect.exe localhost linuxkit
```

## Disks

The Hyper-V backend supports multiple disks to be attached to the VM
using the standard `linuxkit` `-disk` syntax. While Hyper-V typically
stores disk images under a default location, if the VM is created with
`linuxkit`, by default, new disks are created in the current
directory.


## Networking

By default, the Hyper-V backend will try to find an external switch
configured by the user and use this for network connectivity for the
VM.  This means that DHCP will be provided by the normal DHCP server
on your network. Depending on your firewall settings, you may be able
to access the VM directly via its IP address.

Alternatively, you can specify a Hyper-V switch to use using the
`-switch` command-line option. In this case it is the user's
responsibility to provide a DHCP server or to configure the VM's IP
address by some other means.


## Integration services and Metadata

LinuxKit does not yet have packages for Hyper-V integration agents
(KVP and VSS daemons). We plan to add them soon.

The Hyper-V backend currently does not support passing
metadata/userdata to the VM.
