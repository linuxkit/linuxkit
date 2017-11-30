# -*- mode: ruby -*-
# vi: set ft=ruby :
# Author: David Manouchehri

Vagrant.configure("2") do |config|
	config.vm.box = "bento/ubuntu-16.04"

	config.vm.synced_folder ".", "/vagrant", disabled: true

	config.vm.provision "shell", inline: <<-SHELL
		# apt-get update
		# DEBIAN_FRONTEND=noninteractive apt-get -y upgrade
		snap install --classic go
		# apt-get clean
	SHELL

	config.vm.provision "docker"

	config.vm.provision "shell", privileged: false, inline: <<-SHELL
		go get -u github.com/linuxkit/linuxkit/src/cmd/linuxkit
		echo "export PATH=${PATH}:/home/`whoami`/go/bin" >> ~/.bashrc
	SHELL

	%w(vmware_fusion vmware_workstation vmware_appcatalyst).each do |provider|
		config.vm.provider provider do |v|
			v.vmx["vhv.enable"] = "TRUE"
			v.vmx['ethernet0.virtualDev'] = 'vmxnet3'
		end
	end

	config.vm.provider "virtualbox"
	config.vm.provider "hyperv"
end
