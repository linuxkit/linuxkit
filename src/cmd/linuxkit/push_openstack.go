package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

// KeyStoneV3 for OpenStack login
type KeyStoneV3 struct {
	Auth struct {
		Identity struct {
			Methods  []string `json:"methods"`
			Password struct {
				User struct {
					Domain struct {
						Name string `json:"name"`
					} `json:"domain"`
					Name     string `json:"name"`
					Password string `json:"password"`
				} `json:"user"`
			} `json:"password"`
		} `json:"identity"`
		Scope struct {
			Project struct {
				Domain struct {
					Name string `json:"name"`
				} `json:"domain"`
				Name string `json:"name"`
			} `json:"project"`
		} `json:"scope"`
	} `json:"auth"`
}

// GlanceV2Image - the struct for uploading of images
type GlanceV2Image struct {
	ContainerFormat string `json:"container_format"`
	DiskFormat      string `json:"disk_format"`
	Name            string `json:"name"`
}

// GlanceV2ImageResponse - the struct for uploading of images
type GlanceV2ImageResponse struct {
	Status          string        `json:"status"`
	Name            string        `json:"name"`
	Tags            []interface{} `json:"tags"`
	ContainerFormat string        `json:"container_format"`
	CreatedAt       time.Time     `json:"created_at"`
	Size            interface{}   `json:"size"`
	DiskFormat      string        `json:"disk_format"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Visibility      string        `json:"visibility"`
	Locations       []interface{} `json:"locations"`
	Self            string        `json:"self"`
	MinDisk         int           `json:"min_disk"`
	Protected       bool          `json:"protected"`
	ID              string        `json:"id"`
	File            string        `json:"file"`
	Checksum        interface{}   `json:"checksum"`
	Owner           string        `json:"owner"`
	VirtualSize     interface{}   `json:"virtual_size"`
	MinRAM          int           `json:"min_ram"`
	Schema          string        `json:"schema"`
}

// Process the run arguments and execute run
func pushOpenstack(args []string) {
	flags := flag.NewFlagSet("openstack", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s push openstack [options] path\n\n", invoked)
		fmt.Printf("'path' is the full path to an image that will be uploaded to an OpenStack Image store (glance)\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	usernameFlag := flags.String("username", "", "Username with permissions to upload image")
	passwordFlag := flags.String("password", "", "Password for the Username")
	userDomainFlag := flags.String("userDomain", "Default", "")

	projectName := flags.String("project", "", "Name of the Project to be used")
	projectDomain := flags.String("projectDomain", "Default", "")

	keystoneAddress := flags.String("keystoneAddr", "", "The hostname/address of the keystone server to AUTH against, including port(5000)")
	glanceAddress := flags.String("glanceAddr", "", "The hostname/address of the glance server, including port(9292)")

	imageName := flags.String("imageName", "", "A Unique name for the image, if blank the filename will be used")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the path to the image to push\n")
		flags.Usage()
		os.Exit(1)
	}
	filePath := remArgs[0]
	// Check that the file both exists, and can be read
	checkFile(filePath)

	var data KeyStoneV3
	// Defaulting to password login, other login methods may be added later
	data.Auth.Identity.Methods = append(data.Auth.Identity.Methods, "password")

	data.Auth.Identity.Password.User.Name = *usernameFlag
	data.Auth.Identity.Password.User.Domain.Name = *userDomainFlag
	data.Auth.Identity.Password.User.Password = *passwordFlag
	data.Auth.Scope.Project.Domain.Name = *projectDomain
	data.Auth.Scope.Project.Name = *projectName

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("Error building JSON: %v", err)
	}
	body := bytes.NewReader(payloadBytes)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", *keystoneAddress), body)
	if err != nil {
		log.Fatalf("Error Creating HTTP Request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Return Code: %s Error:%v", resp.Status, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 201 { // OK
		// Find the OpenStack KeyStone Token
		openstackToken := resp.Header.Get("x-subject-token")

		log.Debugf("Token => %s", openstackToken)

		if openstackToken == "" {
			log.Fatalln("Error: Can't locate OpenStack Token in Headers")
		}
		// Create a new Image (which will be left in a queued state)
		imageID := createOpenStackImage(filePath, *imageName, *glanceAddress, openstackToken)
		// Take the returned ImageID and upload our image to the created ID
		uploadOpenStackImage(filePath, *glanceAddress, openstackToken, imageID)
	} else {
		message, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("Error authenticating with OpenStack, Error Details:\n%s", string(message))
	}
}

func createOpenStackImage(filePath string, name string, glanceAddress string, token string) string {
	// Currently supported image formats that are both supported by LinuxKit and OpenStack Glance V2
	formats := []string{"ami", "vhd", "vhdx", "vmdk", "raw", "qcow2", "iso"}

	// Find extension of the filename and remove the leading stop
	fileExtension := strings.Replace(path.Ext(filePath), ".", "", -1)
	fileName := strings.TrimSuffix(path.Base(filePath), filepath.Ext(filePath))
	// Check for Supported extension
	var supportedExtension bool
	supportedExtension = false
	for i := 0; i < len(formats); i++ {
		if strings.ContainsAny(fileExtension, formats[i]) {
			supportedExtension = true
		}
	}

	if supportedExtension == false {
		log.Fatalf("Extension [%s] is not supported", fileExtension)
	}

	var image GlanceV2Image
	image.ContainerFormat = "bare"
	image.DiskFormat = fileExtension
	if name == "" {
		image.Name = fileName
	} else {
		image.Name = name
	}

	payloadBytes, err := json.Marshal(image)
	if err != nil {
		log.Fatalf("Error building JSON: %v", err)
	}

	body := bytes.NewReader(payloadBytes)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v2/images", glanceAddress), body)
	if err != nil {
		log.Fatalf("Error Creating HTTP Request:%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Return Code: %s Error:%v", resp.Status, err)
	}
	defer resp.Body.Close()
	var responseJSON GlanceV2ImageResponse
	if resp.StatusCode == 201 { // OK
		bodyBytes, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			log.Fatalf("%v", readErr)
		}

		err = json.Unmarshal(bodyBytes, &responseJSON)
		if err != nil {
			log.Fatalf("%v", err)
		}
		log.Debugf("New Image ID=> %s", responseJSON.ID)

	} else {
		message, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("Error creating new Image [%s], Error: %s", filePath, string(message))
	}
	return responseJSON.ID
}

func uploadOpenStackImage(filePath string, glanceAddress string, token string, imageID string) {

	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v2/images/%s/file", glanceAddress, imageID), f)
	if err != nil {
		log.Fatalf("Error Creating HTTP Request\n%v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Auth-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Return Code: %sError:%v", resp.Status, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 204 { // OK
		log.Infof("Succesfully uploaded [%s]", filePath)
	} else {
		message, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("Error uploading [%s] Error:%s", filePath, string(message))
	}
}
