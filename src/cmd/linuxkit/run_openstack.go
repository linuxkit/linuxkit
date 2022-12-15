package main

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

const (
	defaultOSFlavor = "m1.tiny"
)

func runOpenStackCmd() *cobra.Command {
	var (
		flavorName   string
		instanceName string
		networkID    string
		secGroups    string
		keyName      string
	)

	cmd := &cobra.Command{
		Use:   "openstack",
		Short: "launch an openstack instance using an existing image",
		Long: `Launch an openstack instance using an existing image.
		'name' is the name of an OpenStack image that has already been uploaded using 'linuxkit push'.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run openstack [options] [name]",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if instanceName == "" {
				instanceName = name
			}

			client, err := clientconfig.NewServiceClient("compute", nil)
			if err != nil {
				return fmt.Errorf("Unable to create Compute client, %s", err)
			}

			network := servers.Network{
				UUID: networkID,
			}

			var serverOpts servers.CreateOptsBuilder

			serverOpts = &servers.CreateOpts{
				FlavorName:     flavorName,
				ImageName:      name,
				Name:           instanceName,
				Networks:       []servers.Network{network},
				ServiceClient:  client,
				SecurityGroups: strings.Split(secGroups, ","),
			}

			if keyName != "" {
				serverOpts = &keypairs.CreateOptsExt{
					CreateOptsBuilder: serverOpts,
					KeyName:           keyName,
				}
			}

			server, err := servers.Create(client, serverOpts).Extract()
			if err != nil {
				return fmt.Errorf("Unable to create server: %w", err)
			}

			_ = servers.WaitForStatus(client, server.ID, "ACTIVE", 600)
			log.Infof("Server created, UUID is %s", server.ID)
			fmt.Println(server.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&flavorName, "flavor", defaultOSFlavor, "Instance size (flavor)")
	cmd.Flags().StringVar(&instanceName, "instancename", "", "Name of instance.  Defaults to the name of the image if not specified")
	cmd.Flags().StringVar(&networkID, "network", "", "The ID of the network to attach the instance to")
	cmd.Flags().StringVar(&secGroups, "sec-groups", "default", "Security Group names separated by comma")
	cmd.Flags().StringVar(&keyName, "keyname", "", "The name of the SSH keypair to associate with the instance")

	return cmd
}
