import 'common.rb'

from "linuxkit/mount:ff5338822f20375b8913f5a80f9ed4f6ea9a592b"

script = [
  mount_bind_hostns_self("/etc/cni"), mount_make_hostns_rshared("/etc/cni"),
  mount_bind_hostns_self("/opt/cni"), mount_make_hostns_rshared("/opt/cni"),
  mount_persistent_disk("/var/lib"),
  mkdir_p("/var/lib/kubeadm"),
]

create_shell_wrapper script.join(' && '), '/usr/bin/kube-mounts.sh'
set_exec cmd: [ '/usr/bin/kube-mounts.sh' ]

tag "#{@image_name}:latest-mounts"
