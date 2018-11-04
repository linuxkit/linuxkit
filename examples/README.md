# How to get started with Windows 10

Easiest might be to start with the sshd image. On Windows 10 Professional or Windows 10 Enterprise you can build the image as follows:

`linuxkit.exe build -format iso-efi sshd.yml`

Then you run the image, maybe in Git BASH or some other shell which respects the `$HOME/.ssh/id_rsa.pub` variable:

`linuxkit.exe run -disk size=1 -name=linuxssh sshd-efi.iso`

If all goes well, the linuxkit now boots and you can ssh into linuxkit from another terminal window, using the ip number you see in the host when you run `ifconfig`
