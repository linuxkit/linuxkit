module github.com/linuxkit/linuxkit/src/cmd/linuxkit

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v56.3.0+incompatible
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1
	github.com/Azure/go-autorest v14.2.1-0.20210115164004-c0fe8b0fea3d+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.24
	github.com/Azure/go-autorest/autorest/adal v0.9.18
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Code-Hex/vz v0.0.4
	github.com/Microsoft/go-winio v0.5.2
	github.com/ScaleFT/sshkeys v0.0.0-20181112160850-82451a803681
	github.com/aws/aws-sdk-go v1.44.82
	github.com/containerd/containerd v1.6.6
	github.com/creack/goselect v0.0.0-20180501195510-58854f77ee8d // indirect
	github.com/dchest/bcrypt_pbkdf v0.0.0-20150205184540-83f37f9c154a // indirect
	github.com/docker/buildx v0.8.2
	github.com/docker/cli v20.10.17+incompatible
	github.com/docker/docker v20.10.17+incompatible
	github.com/estesp/manifest-tool/v2 v2.0.6-0.20220728154431-89d791ab7966
	github.com/google/go-containerregistry v0.6.1-0.20211105150418-5c9c442d5d68
	github.com/google/uuid v1.3.0
	github.com/gophercloud/gophercloud v0.1.0
	github.com/gophercloud/utils v0.0.0-20181029231510-34f5991525d1
	github.com/hashicorp/go-version v1.2.0
	github.com/moby/buildkit v0.10.1-0.20220721175135-c75998aec3d4
	github.com/moby/hyperkit v0.0.0-20180416161519-d65b09c1c28a
	//github.com/moby/moby v20.10.3-0.20220728162118-71cb54cec41e+incompatible // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/moby/vpnkit v0.4.1-0.20200311130018-2ffc1dd8a84e
	github.com/moul/gotty-client v1.7.1-0.20180526075433-e5589f6df359
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/packethost/packngo v0.1.1-0.20171201154433-f1be085ecd6f
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5 // indirect
	github.com/pkg/term v1.1.0
	github.com/radu-matei/azure-sdk-for-go v5.0.0-beta.0.20161118192335-3b1282355199+incompatible
	github.com/radu-matei/azure-vhd-utils v0.0.0-20170531165126-e52754d5569d
	github.com/rn/iso9660wrap v0.0.0-20171120145750-baf8d62ad315
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.6
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0 // indirect
	github.com/stretchr/testify v1.7.2
	github.com/surma/gocpio v1.0.2-0.20160926205914-fcb68777e7dc
	github.com/vmware/govmomi v0.20.3
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	golang.org/x/oauth2 v0.0.0-20211005180243-6b3c2da341f1
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad
	google.golang.org/api v0.57.0
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	// these are for the delicate dance of docker/docker, moby/moby, moby/buildkit, estesp/manifest-tool, oras.land/oras-go, linuxkit/linuxkit
	github.com/docker/docker => github.com/moby/moby v20.10.3-0.20220728162118-71cb54cec41e+incompatible
	oras.land/oras-go => oras.land/oras-go v1.1.0
)
