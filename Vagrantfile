# -*- mode: ruby -*-
# vi: set ft=ruby :
# Author: David Manouchehri

Vagrant.configure("2") do |config|
	config.vm.box = "bento/ubuntu-16.04"

	config.vm.provision "shell", inline: <<-SHELL
		# apt-get update
		# DEBIAN_FRONTEND=noninteractive apt-get -y upgrade
		snap install --classic go
		# apt-get clean
	SHELL

	config.vm.provision "docker"

	config.vm.provision "shell", privileged: false, inline: <<-SHELL
		mkdir -p ~/go/src/github.com/linuxkit/
		ln -s /vagrant ~/go/src/github.com/linuxkit/linuxkit
		cd ~/go/src/github.com/linuxkit/linuxkit/src/cmd/linuxkit
		go get -d
		go install
		echo "export PATH=${PATH}:${HOME}/go/bin" >> ~/.bashrc
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
