package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/openstack/clientconfig"

	log "github.com/sirupsen/logrus"
)

const (
	defaultOSFlavor = "m1.tiny"
)

func runOpenStack(args []string) {
	flags := flag.NewFlagSet("openstack", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run openstack [options] [name]\n\n", invoked)
		fmt.Printf("'name' is the name of an OpenStack image that has already been\n")
		fmt.Printf(" uploaded using 'linuxkit push'\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	flavorName := flags.String("flavor", defaultOSFlavor, "Instance size (flavor)")
	instanceName := flags.String("instancename", "", "Name of instance.  Defaults to the name of the image if not specified")
	networkID := flags.String("network", "", "The ID of the network to attach the instance to")
	secGroups := flags.String("sec-groups", "default", "Security Group names separated by comma")
	keyName := flags.String("keyname", "", "The name of the SSH keypair to associate with the instance")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the name of the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	name := remArgs[0]

	if *instanceName == "" {
		*instanceName = name
	}

	client, err := clientconfig.NewServiceClient("compute", nil)
	if err != nil {
		log.Fatalf("Unable to create Compute client, %s", err)
	}

	network := servers.Network{
		UUID: *networkID,
	}

	var serverOpts servers.CreateOptsBuilder

	serverOpts = &servers.CreateOpts{
		FlavorName:     *flavorName,
		ImageName:      name,
		Name:           *instanceName,
		Networks:       []servers.Network{network},
		ServiceClient:  client,
		SecurityGroups: strings.Split(*secGroups, ","),
	}

	if *keyName != "" {
		serverOpts = &keypairs.CreateOptsExt{
			CreateOptsBuilder: serverOpts,
			KeyName:           *keyName,
		}
	}

	server, err := servers.Create(client, serverOpts).Extract()
	if err != nil {
		log.Fatalf("Unable to create server: %s", err)
	}

	servers.WaitForStatus(client, server.ID, "ACTIVE", 600)
	log.Infof("Server created, UUID is %s", server.ID)
	fmt.Println(server.ID)

}
