import 'common.rb'

from "mobylinux/mount:d2669e7c8ddda99fa0618a414d44261eba6e299a"

script = [
  mount_bind_hostns_self("/etc/cni"), mount_make_hostns_rshared("/etc/cni"),
  mount_bind_hostns_self("/opt/cni"), mount_make_hostns_rshared("/opt/cni"),
  mount_persistent_disk("/var/lib"),
  mkdir_p("/var/lib/kubeadm"),
]

create_shell_wrapper script.join(' && '), '/usr/bin/kube-mounts.sh'
set_exec cmd: [ '/usr/bin/kube-mounts.sh' ]

tag "#{@image_name}:latest-mounts"
