FROM fedora:25

RUN dnf install -y 'dnf-command(config-manager)'
RUN dnf config-manager --add-repo \
	http://download.opensuse.org/repositories/home:clearlinux:preview:clear-containers-2.1/Fedora\_25/home:clearlinux:preview:clear-containers-2.1.repo

RUN dnf install -y  qemu-lite clear-containers-image linux-container
COPY qemu.sh /bin/qemu.sh
WORKDIR /root
ENTRYPOINT ["qemu.sh"]
