@image_name = "linuxkit/kubernetes"

@versions = {
  kubernetes: 'v1.6.1',
  weave: 'v1.9.4',
  cni: '0799f5732f2a11b329d9e3d51b9c8f2e3759f2ff',
}

def install_packages pkgs
  cmds = [
    %(apk update),
    %(apk add #{pkgs.join(' ')}),
  ]

  cmds.each { |cmd| run cmd }
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
