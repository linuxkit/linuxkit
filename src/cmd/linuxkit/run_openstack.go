package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

const (
	defaultOSFlavor = "m1.tiny"
	authurlVar      = "OS_AUTH_URL"
	usernameVar     = "OS_USERNAME"
	passwordVar     = "OS_PASSWORD"
	projectNameVar  = "OS_PROJECT_NAME"
	userDomainVar   = "OS_USER_DOMAIN_NAME"
)

func runOpenStack(args []string) {
	flags := flag.NewFlagSet("openstack", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run openstack [options]\n\n", invoked)
		flags.PrintDefaults()
	}
	authurlFlag := flags.String("authurl", "", "The URL of the OpenStack identity service, i.e https://keystone.example.com:5000/v3")
	usernameFlag := flags.String("username", "", "Username with permissions to create an instance")
	passwordFlag := flags.String("password", "", "Password for the specified username")
	projectNameFlag := flags.String("project", "", "Name of the Project (aka Tenant) to be used")
	userDomainFlag := flags.String("domain", "Default", "Domain name")
	imageID := flags.String("img-ID", "", "The ID of the image to boot the instance from")
	networkID := flags.String("network", "", "The ID of the network to attach the instance to")
	flavorName := flags.String("flavor", defaultOSFlavor, "Instance size (flavor)")
	name := flags.String("name", "", "Name of the instance")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	authurl := getStringValue(authurlVar, *authurlFlag, "")
	username := getStringValue(usernameVar, *usernameFlag, "")
	password := getStringValue(passwordVar, *passwordFlag, "")
	projectName := getStringValue(projectNameVar, *projectNameFlag, "")
	userDomain := getStringValue(userDomainVar, *userDomainFlag, "")

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: authurl,
		Username:         username,
		Password:         password,
		DomainName:       userDomain,
		TenantName:       projectName,
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		log.Fatalf("Failed to authenticate")
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		log.Fatalf("Unable to create Compute V2 client, %s", err)
	}

	network := servers.Network{
		UUID: *networkID,
	}

	serverOpts := &servers.CreateOpts{
		FlavorName:    *flavorName,
		ImageRef:      *imageID,
		Name:          *name,
		Networks:      []servers.Network{network},
		ServiceClient: client,
	}

	server, err := servers.Create(client, serverOpts).Extract()
	if err != nil {
		log.Fatalf("Unable to create server: %s", err)
	}

	servers.WaitForStatus(client, server.ID, "ACTIVE", 600)
	fmt.Printf("Server created, UUID is %s", server.ID)

}
