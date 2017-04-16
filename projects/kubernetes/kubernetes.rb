import 'common.rb'

from "alpine:edge"

def install_node_dependencies
  kube_release_artefacts = "https://dl.k8s.io/#{@versions[:kubernetes]}/bin/linux/amd64"
  cni_release_artefacts = "https://dl.k8s.io/network-plugins/cni-amd64-#{@versions[:cni]}.tar.gz"
  weave_launcher = "https://cloud.weave.works/k8s/v1.6/net?v=#{@versions[:weave]}"

  download_files = [
    '/etc/weave.yaml' => {
      url: weave_launcher,
      mode: '0644',
    },
    '/tmp/cni.tgz' => {
      url: cni_release_artefacts,
      mode: '0644',
    },
    '/usr/bin/kubelet' => {
      url: "#{kube_release_artefacts}/kubelet",
      mode: '0755',
    },
    '/usr/bin/kubeadm' => {
      url: "#{kube_release_artefacts}/kubeadm",
      mode: '0755',
    },
    '/usr/bin/kubectl' => {
      url: "#{kube_release_artefacts}/kubectl",
      mode: '0755',
    },
  ]

  download_files.each do |file|
    file.each do |dest,info|
      run %(curl --output "#{dest}" --fail --silent --location "#{info[:url]}")
      run %(chmod "#{info[:mode]}" "#{dest}")
    end
  end

  run "mkdir -p /opt/cni/bin /etc/cni/net.d && tar xzf /tmp/cni.tgz -C /opt/cni && rm -f /tmp/cni.tgz"
end

def kubelet_cmd
  %w(
    kubelet
      --kubeconfig=/var/lib/kubeadm/kubelet.conf --require-kubeconfig=true
      --pod-manifest-path=/var/lib/kubeadm/manifests --allow-privileged=true
      --cluster-dns=10.96.0.10 --cluster-domain=cluster.local
      --cgroups-per-qos=false --enforce-node-allocatable=""
      --network-plugin=cni --cni-conf-dir=/etc/cni/net.d --cni-bin-dir=/opt/cni/bin
  )
end

kubelet_dependencies = %w(libc6-compat util-linux iproute2 iptables ebtables ethtool socat curl)
install_packages kubelet_dependencies
install_node_dependencies

# Exploit shared mounts, give CNI paths back to the host
mount_cni_dirs = [
  mount_bind("/opt/cni", "/rootfs/opt/cni"),
  mount_bind("/etc/cni", "/rootfs/etc/cni"),
]

# At the moment we trigger `kubeadm init` manually on the master, then start nodes which expect `kubeadm join` args in metadata volume
wait_for_node_metadata_or_sleep_until_master_init = "[ ! -e /dev/sr0 ] && sleep 1 || (mount -o ro /dev/sr0 /mnt && kubeadm join --skip-preflight-checks \\\$(cat /mnt/config))"

create_shell_wrapper "#{mount_cni_dirs.join(' && ')} && until #{kubelet_cmd.join(' ')} ; do #{wait_for_node_metadata_or_sleep_until_master_init} ; done", '/usr/bin/kubelet.sh'

create_shell_wrapper "kubeadm init --skip-preflight-checks --kubernetes-version #{@versions[:kubernetes]} && kubectl create -n kube-system -f /etc/weave.yaml", '/usr/bin/kubeadm-init.sh'

flatten

env KUBECONFIG: "/etc/kubernetes/admin.conf"

set_exec entrypoint: %w(kubelet.sh)

tag "#{@image_name}:latest"
