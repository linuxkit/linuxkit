# Initial install

I use Windows 10 Pro in a Vmware Fusion VM. Make sure it's a 64bit VM.

In Vmware Fusion VM settings make sure the `Processor -> Advanced Options -> Enable hypervisor applications in virtual machine` is selected. This enables nested virtualisation. If you install on bare hardware make sure the virtualisation technology is enabled in the BIOS.

When creating the user, make sure that te username does **not** contain
any spaces. This will save you a world of pain!  On Windows 10, also select custom settings during the install and disable all the spying/calling home features introduced. Also, since I'm running Windows in a VM on an already password protected system, I disable password for my user, using `c:\Windows\System32\netplwiz.exe`. Just untick the password checkbox.


Install software:
- [Git](http://git-scm.com/): Make sure you select 'Use git from Windows command prompt'. It gives access to git from PS, but still installs git-bash.
- [Putty](http://www.chiark.greenend.org.uk/~sgtatham/putty/download.html). For getting to the serial console of the MobyLinux VM.
- [Sysinternals](https://technet.microsoft.com/en-gb/sysinternals/bb842062). Generally useful.
- [Chocolatey](https://chocolatey.org/). It's kinda like homebrew for windows.


## Enable Hyper-V feature

This [MSDN article](https://msdn.microsoft.com/en-us/virtualization/hyperv_on_windows/quick_start/walkthrough_install) is useful.

Install Hyper-V with powershell:
```
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V –All
```
This did *not* work! So I had to use the GUI to install instead as per
the MSDN article.


# Boot MobyLinux

## Create a MobyLinux ISO image

For now, this has to be done on a Linux docker install.

Clone the [Moby git repository](https://github.com:docker/moby.git), cd into it and then do:
```
cd alpine
make mobylinux.iso
```
Copy the iso image to your Windows host/VM.


## Create a switch

We need to create a VM Switch to attach the MobyLinux Networking to.  This is a one off operation.  Check your main Ethernet interface with either `ipconfig` or `Get-NetAdapter` (in powershell).  On my system it is called 'Ethernet0'. Then create a switch like this (in elevated powershell):

```
New-VMSwitch -Name "VirtualSwitch" -AllowManagementOS $True  -NetAdapterName Ethernet0
```
TODO: Figure out how to configure a NAT switch


## Booting MobyLinux from ISO

In the MobyLinux repository there is a Powershell script called `MobyLinux.ps1` which allows you to create, start, stop and destroy a MobyLinux VM.  Copy it over to your Windows machine.

This must be executed from an elevated Powershell (ie Run as Administrator).

Some Windows installation may not allow execution of arbitrary Powershell scripts.  Check with `Get-ExecutionPolicy`. It is likely set to 'Restricted', which prevents you from running scripts. Change the policy:
```
Set-ExecutionPolicy -ExecutionPolicy Unrestricted
```

Now, you can create and start a new MobyLinux VM

```
.\MobyLinux.ps1 -IsoFile .\mobylinux.iso -create -start
```

You can stop the VM with:
```
.\MobyLinux.ps1 -stop
```
and it can be restarted with:
```
.\MobyLinux.ps1 -start
```
and all the files can be removed with:
```
.\MobyLinux.ps1 -destroy
```


## Getting a serial console

The MobyLinux VM is configured with a serial console which Hyper-V relays to a named pipe.  You can attach putty to the named pipe to get the console:
```
'C:\Program Files (x86)\PuTTY\putty.exe' -serial \\.\pipe\MobyLinuxVM-com1
```

For easy access, I create a shortcut to putty.exe on my Desktop,
rename it to "MobyVM Console", open the shortcur properties and cut
and paste the above line into the "Target" field in the Shortcut tab. Note, the shortcut needs to be executed as Administrator to work.


# ToDos and Open issues
- Networking configuration
  - switch between wifi/wired (see also below)
  - NAT, see eg the Docker on Windows docs on MSDN
- Host FS sharing (SMB?)
- Host <-> docker in VM communication (with Proxy)
  - maybe hijack serial console as transport...
- Start Hyper-V guest services in Moby
- Would like to use a Hyper-V generation 2 VM. That requires and ISO
  with UEFI boot (see below). Though, Azure might currently only
  support Generation 1 VMs
- Logging

many more

```
If running Windows 10 Hyper-V on a laptop you may want to create a
virtual switch for both the ethernet and wireless network cards. With
this configuration you can change your virtual machines between theses
switches dependent on how the laptop is network connected. Virtual
machines will not automatically switch between wired and wireless.
```

# Notes, thoughts, links

[Serial Console Service](https://github.com/alexpilotti/SerialConsoleService): This is a Windows Service (written in C#) running inside a Windows VM opening a command prompt if something attaches to the named pipe on the the host.  Might be useful for Host<->VM communication.

[Generation 2 VMs](https://blogs.technet.microsoft.com/jhoward/2013/10/24/hyper-v-generation-2-virtual-machines-part-1/): Contains very useful details about differences between generation 1 and generation 2 Hyper-V VMs.

Thought: Another option for booting Moby on Hyper-V might be to boot a EFI file from Hyper-V and have that directly load the Linux kernel and initrd like we kinda do on xhyve. Would safe us the whole ISO image hassle.

[Create a Windows VM to run Containers](https://msdn.microsoft.com/en-us/virtualization/windowscontainers/quick_start/container_setup): Contains good script for config

[Enable Windows server to run Containers](https://msdn.microsoft.com/en-us/virtualization/windowscontainers/quick_start/inplace_setup): Another good script linked fro here for setting up Networking.

[Shared Network setup](http://blog.areflyen.no/2012/10/10/setting-up-internet-access-for-hyper-v-with-nat-in-windows-8/): GUI, maybe convert to PS.

## UEFI Boot
Hyper-V Generation 2 VMs require UEFI booting. I've tried many
different ways to create Hybrid UEFI/Legacy boot ISO images and none
of them worked...Might need to restart the effort at some point.

Maybe just create a FAT32 formatted raw disk, copy EFI linux loader
(efilinux, from kernel source tree) to it along with the kernel image and initrd (no
syslinux etc). Roughly following
[this guide](https://wiki.ubuntu.com/USBStickUEFIHowto), but not
bother with the main partition. Convert this to a VHD(X) using the
vbox script (or maybe on windows with appropriate tools) and boot from
there. [Enterprise](https://sevenbits.github.io/Enterprise) might be
an alternative Linux UEFI loader.

Another alternative would be to boot a Linux EFI loader from Hyper-V
and pass enough arguments for it to load kernel and initrd directly
from the host file system.

Vmware Fusion UEFI boot. Add `firmware = "efi"` to the `.vmx` file.
