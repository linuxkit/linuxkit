@image_name = "mobylinux/kubernetes"

@versions = {
  kubernetes: "v1.6.1",
  weave: "v1.9.4",
  tini: "v0.14.0",
}

def install_packages pkgs
  cmds = [
    %(apt-get update -q),
    %(apt-get upgrade -qy),
    %(apt-get install -qy #{pkgs}),
  ]

  cmds.each { |cmd| run cmd }
end

def setup_apt_config
  prepare = [
    'curl --silent "https://packages.cloud.google.com/apt/doc/apt-key.gpg" | apt-key add -',
    'echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list',
  ]

  dependencies = %(curl apt-transport-https)

  install_packages dependencies

  prepare.each { |cmd| run cmd }
end

def create_shell_wrapper script, path
  run "echo \"#!/bin/sh\n#{script}\n\" > #{path} && chmod 0755 #{path}"
end

def mount_bind src, dst
  "mount --bind #{src} #{dst}"
end

def mount_bind_hostns_self mnt
  "nsenter --mount=/proc/1/ns/mnt mount -- --bind #{mnt} #{mnt}"
end

def mount_make_hostns_rshared mnt
  "nsenter --mount=/proc/1/ns/mnt mount -- --make-rshared #{mnt}"
end

def mount_persistent_disk mnt
  "/mount.sh #{mnt}"
end

def mkdir_p dir
  "mkdir -p #{dir}"
end
