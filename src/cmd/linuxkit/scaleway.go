package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	gotty "github.com/moul/gotty-client"
	scw "github.com/scaleway/go-scaleway"
	"github.com/scaleway/go-scaleway/logger"
	"github.com/scaleway/go-scaleway/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// ScalewayClient contains state required for communication with Scaleway as well as the instance
type ScalewayClient struct {
	api       *scw.ScalewayAPI
	fileName  string
	region    string
	sshConfig *ssh.ClientConfig
}

// ScalewayConfig contains required field to read scaleway config file
type ScalewayConfig struct {
	Organization string `json:"organization"`
	Token        string `json:"token"`
	Version      string `json:"version"`
}

// NewScalewayClient creates a new scaleway client
func NewScalewayClient(token, region string) (*ScalewayClient, error) {
	log.Debugf("Connecting to Scaleway")
	organization := ""
	if token == "" {
		log.Debugf("Using .scwrc file to get token")
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE") // Windows support
		}
		if homeDir == "" {
			return nil, fmt.Errorf("Home directory not found")
		}
		swrcPath := filepath.Join(homeDir, ".scwrc")

		file, err := ioutil.ReadFile(swrcPath)
		if err != nil {
			return nil, fmt.Errorf("Error reading Scaleway config file: %v", err)
		}

		var scalewayConfig ScalewayConfig
		err = json.Unmarshal(file, &scalewayConfig)
		if err != nil {
			return nil, fmt.Errorf("Error during unmarshal of Scaleway config file: %v", err)
		}

		token = scalewayConfig.Token
		organization = scalewayConfig.Organization
	}

	api, err := scw.NewScalewayAPI(organization, token, "", region)
	if err != nil {
		return nil, err
	}

	l := logger.NewDisableLogger()
	api.Logger = l

	if organization == "" {
		organisations, err := api.GetOrganization()
		if err != nil {
			return nil, err
		}
		api.Organization = organisations.Organizations[0].ID
	}

	client := &ScalewayClient{
		api:      api,
		fileName: "",
		region:   region,
	}

	return client, nil
}

// CreateInstance create an instance with one additional volume
func (s *ScalewayClient) CreateInstance() (string, error) {
	// get the Ubuntu Xenial image id
	image, err := s.api.GetImageID("Ubuntu Xenial", "x86_64") // TODO fix arch and use from args
	if err != nil {
		return "", err
	}
	imageID := image.Identifier

	var serverDefinition types.ScalewayServerDefinition

	serverDefinition.Name = "linuxkit-builder"
	serverDefinition.Image = &imageID
	serverDefinition.CommercialType = "VC1M" // TODO use args?

	// creation of second volume
	var volumeDefinition types.ScalewayVolumeDefinition
	volumeDefinition.Name = "linuxkit-builder-volume"
	volumeDefinition.Size = 50000000000 // FIX remove hardcoded value
	volumeDefinition.Type = "l_ssd"

	log.Debugf("Creating volume on Scaleway")
	volumeID, err := s.api.PostVolume(volumeDefinition)
	if err != nil {
		return "", err
	}

	serverDefinition.Volumes = make(map[string]string)
	serverDefinition.Volumes["1"] = volumeID

	serverID, err := s.api.PostServer(serverDefinition)
	if err != nil {
		return "", err
	}

	log.Debugf("Created server %s on Scaleway", serverID)
	return serverID, nil
}

// GetSecondVolumeID returns the ID of the second volume of the server
func (s *ScalewayClient) GetSecondVolumeID(instanceID string) (string, error) {
	server, err := s.api.GetServer(instanceID)
	if err != nil {
		return "", err
	}

	secondVolume, ok := server.Volumes["1"]
	if !ok {
		return "", errors.New("No second volume found")
	}

	return secondVolume.Identifier, nil
}

// BootInstanceAndWait boots and wait for instance to be booted
func (s *ScalewayClient) BootInstanceAndWait(instanceID string) error {
	err := s.api.PostServerAction(instanceID, "poweron")
	if err != nil {
		return err
	}

	log.Debugf("Waiting for server %s to be started", instanceID)

	// code taken from scaleway-cli, could need some changes
	promise := make(chan bool)
	var server *types.ScalewayServer
	var currentState string

	go func() {
		defer close(promise)

		for {
			server, err = s.api.GetServer(instanceID)
			if err != nil {
				promise <- false
				return
			}

			if currentState != server.State {
				currentState = server.State
			}

			if server.State == "running" {
				break
			}
			if server.State == "stopped" {
				promise <- false
				return
			}
			time.Sleep(1 * time.Second)
		}

		ip := server.PublicAddress.IP
		if ip == "" && server.EnableIPV6 {
			ip = fmt.Sprintf("[%s]", server.IPV6.Address)
		}
		dest := fmt.Sprintf("%s:22", ip)
		for {
			conn, err := net.Dial("tcp", dest)
			if err == nil {
				defer conn.Close()
				break
			} else {
				time.Sleep(1 * time.Second)
			}
		}
		promise <- true
	}()

	loop := 0
	for {
		select {
		case done := <-promise:
			if !done {
				return err
			}
			log.Debugf("Server %s started", instanceID)
			return nil
		case <-time.After(time.Millisecond * 100):
			loop = loop + 1
			if loop == 5 {
				loop = 0
			}
		}
	}
}

// getSSHAuth is uses to get the ssh.Signer needed to connect via SSH
func getSSHAuth(sshKeyPath string) (ssh.Signer, error) {
	f, err := os.Open(sshKeyPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}
	return signer, err
}

// CopyImageToInstance copies the image to the instance via ssh
func (s *ScalewayClient) CopyImageToInstance(instanceID, path, sshKeyPath string) error {
	_, base := filepath.Split(path)
	s.fileName = base

	server, err := s.api.GetServer(instanceID)
	if err != nil {
		return err
	}

	signer, err := getSSHAuth(sshKeyPath)
	if err != nil {
		return err
	}

	s.sshConfig = &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO validate server before?
	}

	client, err := ssh.Dial("tcp", server.PublicAddress.IP+":22", s.sshConfig) // TODO remove hardocoded port?
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// code taken from bramvdbogaerde/go-scp
	contentBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	bytesReader := bytes.NewReader(contentBytes)

	log.Infof("Starting to upload %s on server", base)

	go func() {
		w, err := session.StdinPipe()
		if err != nil {
			return
		}
		defer w.Close()
		fmt.Fprintln(w, "C0600", int64(len(contentBytes)), base)
		io.Copy(w, bytesReader)
		fmt.Fprintln(w, "\x00")
	}()

	session.Run("/usr/bin/scp -t /root/") // TODO remove hardcoded remote path?
	return err
}

// WriteImageToVolume does a dd command on the remote instance via ssh
func (s *ScalewayClient) WriteImageToVolume(instanceID, deviceName string) error {
	server, err := s.api.GetServer(instanceID)
	if err != nil {
		return err
	}

	client, err := ssh.Dial("tcp", server.PublicAddress.IP+":22", s.sshConfig) // TODO remove hardcoded port + use the same dial as before?
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var ddPathBuf bytes.Buffer
	session.Stdout = &ddPathBuf

	err = session.Run("which dd") // get the right path
	if err != nil {
		return err
	}

	session, err = client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	ddCommand := strings.Trim(ddPathBuf.String(), " \n")
	command := fmt.Sprintf("%s if=%s of=%s", ddCommand, s.fileName, deviceName)

	log.Infof("Starting writing iso to disk")

	err = session.Run(command)
	if err != nil {
		return err
	}

	log.Infof("ISO image written to disk")

	return nil
}

// TerminateInstance terminates the instance and wait for termination
func (s *ScalewayClient) TerminateInstance(instanceID string) error {
	server, err := s.api.GetServer(instanceID)
	if err != nil {
		return err
	}

	log.Debugf("Shutting down server %s", instanceID)

	err = s.api.PostServerAction(server.Identifier, "poweroff")
	if err != nil {
		return err
	}

	// code taken from scaleway-cli
	time.Sleep(10 * time.Second)

	var currentState string

	log.Debugf("Waiting for server to shutdown")

	for {
		server, err = s.api.GetServer(instanceID)
		if err != nil {
			return err
		}
		if currentState != server.State {
			currentState = server.State
		}
		if server.State == "stopped" {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// CreateScalewayImage creates the image and delete old image and snapshot if same name
func (s *ScalewayClient) CreateScalewayImage(instanceID, volumeID, name string) error {
	oldImage, err := s.api.GetImageID(name, "x86_64")
	if err == nil {
		err = s.api.DeleteImage(oldImage.Identifier)
		if err != nil {
			return err
		}
	}

	oldSnapshot, err := s.api.GetSnapshotID(name)
	if err == nil {
		err := s.api.DeleteSnapshot(oldSnapshot)
		if err != nil {
			return err
		}
	}

	snapshotID, err := s.api.PostSnapshot(volumeID, name)
	if err != nil {
		return err
	}

	imageID, err := s.api.PostImage(snapshotID, name, "", "x86_64") // TODO remove hardcoded arch
	if err != nil {
		return err
	}

	log.Infof("Image %s with ID %s created", name, imageID)

	return nil
}

// DeleteInstanceAndVolumes deletes the instance and the volumes attached
func (s *ScalewayClient) DeleteInstanceAndVolumes(instanceID string) error {
	server, err := s.api.GetServer(instanceID)
	if err != nil {
		return err
	}

	err = s.api.DeleteServer(instanceID)
	if err != nil {
		return err
	}

	for _, volume := range server.Volumes {
		err = s.api.DeleteVolume(volume.Identifier)
		if err != nil {
			return err
		}
	}

	log.Infof("Server %s deleted", instanceID)

	return nil
}

// CreateLinuxkitInstance creates an instance with the given linuxkit image
func (s *ScalewayClient) CreateLinuxkitInstance(instanceName, imageName, instanceType string) (string, error) {
	// get the image ID
	image, err := s.api.GetImageID(imageName, "x86_64") // TODO fix arch and use from args
	if err != nil {
		return "", err
	}
	imageID := image.Identifier

	var serverDefinition types.ScalewayServerDefinition

	serverDefinition.Name = instanceName
	serverDefinition.Image = &imageID
	serverDefinition.CommercialType = instanceType
	serverDefinition.BootType = "local"

	log.Debugf("Creating volume on Scaleway")

	log.Debugf("Creating server %s on Scaleway", serverDefinition.Name)
	serverID, err := s.api.PostServer(serverDefinition)
	if err != nil {
		return "", err
	}

	return serverID, nil
}

// BootInstance boots the specified instance, and don't wait
func (s *ScalewayClient) BootInstance(instanceID string) error {
	err := s.api.PostServerAction(instanceID, "poweron")
	if err != nil {
		return err
	}
	return nil
}

// ConnectSerialPort connects to the serial port of the instance
func (s *ScalewayClient) ConnectSerialPort(instanceID string) error {
	var gottyURL string
	switch s.region {
	case "par1":
		gottyURL = "https://tty-par1.scaleway.com/v2/"
	case "ams1":
		gottyURL = "https://tty-ams1.scaleway.com/"
	default:
		return errors.New("Instance have no region")
	}

	fullURL := fmt.Sprintf("%s?arg=%s&arg=%s", gottyURL, s.api.Token, instanceID)

	log.Debugf("Connection to ", fullURL)
	gottyClient, err := gotty.NewClient(fullURL)
	if err != nil {
		return err
	}

	gottyClient.SkipTLSVerify = true

	gottyClient.UseProxyFromEnv = true

	err = gottyClient.Connect()
	if err != nil {
		return err
	}

	done := make(chan bool)

	fmt.Println("You are connected, type 'Ctrl+q' to quit.")
	go func() {
		err = gottyClient.Loop()
		if err != nil {
			fmt.Printf("ERROR: " + err.Error())
		}
		//gottyClient.Close()
		done <- true
	}()
	<-done
	return nil
}
