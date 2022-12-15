package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/soap"
)

func pushVCenterCmd() *cobra.Command {
	var (
		url        string
		datacenter string
		datastore  string
		hostname   string
		folder     string
	)
	cmd := &cobra.Command{
		Use:   "vcenter",
		Short: "push image to Azure",
		Long: `Push image to Azure.
		First argument specifies the full path of an ISO image. It will be pushed to a vCenter cluster.
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filepath := args[0]

			newVM := vmConfig{
				vCenterURL:  &url,
				dcName:      &datacenter,
				dsName:      &datastore,
				vSphereHost: &hostname,
				path:        &filepath,
				vmFolder:    &folder,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// Ensure an iso has been passed to the vCenter push Command
			if !strings.HasSuffix(*newVM.path, ".iso") {
				log.Fatalln("Please specify an '.iso' file")
			}

			// Test any passed in files before uploading image
			checkFile(*newVM.path)

			// Connect to VMware vCenter and return the values needed to upload image
			c, dss, _, _, _, _ := vCenterConnect(ctx, newVM)

			// Create a folder from the uploaded image name if needed
			if *newVM.vmFolder == "" {
				*newVM.vmFolder = strings.TrimSuffix(path.Base(*newVM.path), ".iso")
			}

			// The CreateFolder method isn't necessary as the *newVM.vmname will be created automatically
			uploadFile(c, newVM, dss)

			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", os.Getenv("VCURL"), "URL of VMware vCenter in the format of https://username:password@VCaddress/sdk")
	cmd.Flags().StringVar(&datacenter, "datacenter", os.Getenv("VCDATACENTER"), "The name of the DataCenter to host the image")
	cmd.Flags().StringVar(&datastore, "datastore", os.Getenv("VCDATASTORE"), "The name of the DataStore to host the image")
	cmd.Flags().StringVar(&hostname, "hostname", os.Getenv("VCHOST"), "The server that will host the image")
	cmd.Flags().StringVar(&folder, "folder", "", "A folder on the datastore to push the image too")

	return cmd
}

func checkFile(file string) {
	if _, err := os.Stat(file); err != nil {
		if os.IsPermission(err) {
			log.Fatalf("Unable to read file [%s], please check permissions", file)
		} else if os.IsNotExist(err) {
			log.Fatalf("File [%s], does not exist", file)
		} else {
			log.Fatalf("Unable to stat file [%s]: %v", file, err)
		}
	}
}

func uploadFile(c *govmomi.Client, newVM vmConfig, dss *object.Datastore) {
	_, fileName := path.Split(*newVM.path)
	log.Infof("Uploading LinuxKit file [%s]", *newVM.path)
	if *newVM.path == "" {
		log.Fatalf("No file specified")
	}
	dsurl := dss.NewURL(fmt.Sprintf("%s/%s", *newVM.vmFolder, fileName))

	p := soap.DefaultUpload
	ctx := context.Background()
	if err := c.Client.UploadFile(ctx, *newVM.path, dsurl, &p); err != nil {
		log.Fatalf("Unable to upload file to vCenter Datastore\n%v", err)
	}
}
