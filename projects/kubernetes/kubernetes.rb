import 'common.rb'

from "gcr.io/google_containers/hyperkube-amd64:#{@versions[:kubernetes]}"

def install_node_dependencies
  kube_release_artefacts = "https://dl.k8s.io/#{@versions[:kubernetes]}/bin/linux/amd64"
  weave_launcher = "https://frontend.dev.weave.works/k8s/v1.6/net?v=#{@versions[:weave]}"

  download_files = [
    "/etc/weave.yaml" => {
      url: weave_launcher,
      mode: '0644',
    },
    "/usr/bin/kubeadm" => {
      url: "#{kube_release_artefacts}/kubeadm",
      mode: '0755',
    },
    "/usr/bin/tini" => {
      url: "https://github.com/krallin/tini/releases/download/#{@versions[:tini]}/tini",
      mode: '0755',
    },
  ]

  download_files.each do |file|
    file.each do |dest,info|
      run %(curl --insecure --output "#{dest}" --fail --silent --location "#{info[:url]}")
      run %(chmod "#{info[:mode]}" "#{dest}")
    end
  end
end

def kubelet_cmd
  %w(
    /hyperkube kubelet
      --kubeconfig=/var/lib/kubeadm/kubelet.conf --require-kubeconfig=true
      --pod-manifest-path=/var/lib/kubeadm/manifests --allow-privileged=true
      --cluster-dns=10.96.0.10 --cluster-domain=cluster.local
      --cgroups-per-qos=false --enforce-node-allocatable=""
      --network-plugin=cni --cni-conf-dir=/etc/cni/net.d --cni-bin-dir=/opt/cni/bin
  )
  #--node-ip="192.168.65.2"
end

setup_apt_config
run "rm -f /etc/cni/net.d/10-containernet.conf"
install_packages 'kubernetes-cni'
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

set_exec entrypoint: %w(tini -s --), cmd: %w(kubelet.sh)

tag "#{@image_name}:latest"
