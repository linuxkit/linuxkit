package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ScaleFT/sshkeys"
	gotty "github.com/moul/gotty-client"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/api/marketplace/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

var (
	defaultScalewayCommercialType          = "DEV1-S"
	defaultScalewayImageName               = "Ubuntu Bionic"
	defaultScalewayImageArch               = "x86_64"
	scalewayDynamicIPRequired              = true
	scalewayBootType                       = instance.BootTypeLocal
	scalewayInstanceVolumeSize    scw.Size = 20
)

// ScalewayClient contains state required for communication with Scaleway as well as the instance
type ScalewayClient struct {
	instanceAPI    *instance.API
	marketplaceAPI *marketplace.API
	fileName       string
	zone           scw.Zone
	sshConfig      *ssh.ClientConfig
	secretKey      string
}

// NewScalewayClient creates a new scaleway client
func NewScalewayClient(accessKey, secretKey, zone, organizationID string) (*ScalewayClient, error) {
	log.Debugf("Connecting to Scaleway")

	var scwOptions []scw.ClientOption
	if accessKey == "" || secretKey == "" {
		config, err := scw.LoadConfig()
		if err != nil {
			return nil, err
		}
		profile, err := config.GetActiveProfile()
		if err != nil {
			return nil, err
		}

		scwOptions = append(scwOptions, scw.WithProfile(profile), scw.WithEnv())
		if *profile.DefaultZone != "" {
			zone = *profile.DefaultZone
		}
	} else {
		scwOptions = append(
			scwOptions,
			scw.WithAuth(accessKey, secretKey),
			scw.WithDefaultOrganizationID(organizationID),
		)
	}

	scwZone, err := scw.ParseZone(zone)
	if err != nil {
		return nil, err
	}
	scwOptions = append(scwOptions, scw.WithDefaultZone(scwZone))

	scwClient, err := scw.NewClient(scwOptions...)
	if err != nil {
		return nil, err
	}
	instanceAPI := instance.NewAPI(scwClient)
	marketplaceAPI := marketplace.NewAPI(scwClient)

	client := &ScalewayClient{
		instanceAPI:    instanceAPI,
		marketplaceAPI: marketplaceAPI,
		zone:           scwZone,
		fileName:       "",
		secretKey:      secretKey,
	}

	return client, nil
}

func (s *ScalewayClient) getImageID(imageName, commercialType, arch string) (string, error) {
	imagesResp, err := s.marketplaceAPI.ListImages(&marketplace.ListImagesRequest{})
	if err != nil {
		return "", err
	}
	for _, image := range imagesResp.Images {
		if image.Name == imageName {
			for _, version := range image.Versions {
				for _, localImage := range version.LocalImages {
					if localImage.Arch == arch && localImage.Zone == s.zone {
						for _, compatibleCommercialType := range localImage.CompatibleCommercialTypes {
							if compatibleCommercialType == commercialType {
								return localImage.ID, nil
							}
						}
					}
				}
			}
		}
	}
	return "", errors.New("No image matching given requests")
}

// CreateInstance create an instance with one additional volume
func (s *ScalewayClient) CreateInstance(volumeSize int) (string, error) {
	// get the Ubuntu Bionic image id
	imageID, err := s.getImageID(defaultScalewayImageName, defaultScalewayCommercialType, defaultScalewayImageArch)
	if err != nil {
		return "", err
	}

	scwVolumeSize := scw.Size(volumeSize)
	builderVolumeSize := scwVolumeSize * scw.GB

	createVolumeRequest := &instance.CreateVolumeRequest{
		Name:       "linuxkit-builder-volume",
		VolumeType: "l_ssd",
		Size:       &builderVolumeSize,
	}

	log.Debugf("Creating volume on Scaleway")
	volumeResp, err := s.instanceAPI.CreateVolume(createVolumeRequest)
	if err != nil {
		return "", err
	}

	volumeMap := map[string]*instance.VolumeTemplate{
		"0": {Size: (scalewayInstanceVolumeSize - scwVolumeSize) * scw.GB},
	}

	createServerRequest := &instance.CreateServerRequest{
		Name:              "linuxkit-builder",
		CommercialType:    defaultScalewayCommercialType,
		DynamicIPRequired: &scalewayDynamicIPRequired,
		Image:             imageID,
		EnableIPv6:        false,
		BootType:          &scalewayBootType,
		Volumes:           volumeMap,
	}

	log.Debug("Creating server on Scaleway")
	serverResp, err := s.instanceAPI.CreateServer(createServerRequest)
	if err != nil {
		return "", err
	}

	attachVolumeRequest := &instance.AttachVolumeRequest{
		ServerID: serverResp.Server.ID,
		VolumeID: volumeResp.Volume.ID,
	}

	_, err = s.instanceAPI.AttachVolume(attachVolumeRequest)
	if err != nil {
		return "", nil
	}

	log.Debugf("Created server %s on Scaleway", serverResp.Server.ID)
	return serverResp.Server.ID, nil
}

// GetSecondVolumeID returns the ID of the second volume of the server
func (s *ScalewayClient) GetSecondVolumeID(instanceID string) (string, error) {
	getServerRequest := &instance.GetServerRequest{
		ServerID: instanceID,
	}

	serverResp, err := s.instanceAPI.GetServer(getServerRequest)
	if err != nil {
		return "", err
	}

	secondVolume, ok := serverResp.Server.Volumes["1"]
	if !ok {
		return "", errors.New("No second volume found")
	}

	return secondVolume.ID, nil
}

// BootInstanceAndWait boots and wait for instance to be booted
func (s *ScalewayClient) BootInstanceAndWait(instanceID string) error {
	serverActionRequest := &instance.ServerActionRequest{
		ServerID: instanceID,
		Action:   instance.ServerActionPoweron,
	}

	_, err := s.instanceAPI.ServerAction(serverActionRequest)
	if err != nil {
		return err
	}

	log.Debugf("Waiting for server %s to be started", instanceID)

	// code taken from scaleway-cli, could need some changes
	promise := make(chan bool)
	var server *instance.Server
	var currentState instance.ServerState

	go func() {
		defer close(promise)

		for {
			serverResp, err := s.instanceAPI.GetServer(&instance.GetServerRequest{
				ServerID: instanceID,
			})
			server = serverResp.Server
			if err != nil {
				promise <- false
				return
			}

			if currentState != server.State {
				currentState = server.State
			}

			if server.State == instance.ServerStateRunning {
				break
			}
			if server.State == instance.ServerStateStopped {
				promise <- false
				return
			}
			time.Sleep(1 * time.Second)
		}

		ip := server.PublicIP.Address.String()
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

// getSSHAuth gets the ssh.Signer needed to connect via SSH
func getSSHAuth(sshKeyPath string) (ssh.Signer, error) {
	f, err := os.Open(sshKeyPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		fmt.Print("Enter ssh key passphrase: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		// ReadPassword eats newline, put it back to avoid mangling logs
		fmt.Println()
		if err != nil {
			return nil, err
		}
		signer, err := sshkeys.ParseEncryptedPrivateKey(buf, bytePassword)
		if err != nil {
			return nil, err
		}
		return signer, nil
	}
	return signer, err
}

// CopyImageToInstance copies the image to the instance via ssh
func (s *ScalewayClient) CopyImageToInstance(instanceID, path, sshKeyPath string) error {
	_, base := filepath.Split(path)
	s.fileName = base

	serverResp, err := s.instanceAPI.GetServer(&instance.GetServerRequest{
		ServerID: instanceID,
	})
	if err != nil {
		return err
	}
	server := serverResp.Server

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

	client, err := ssh.Dial("tcp", server.PublicIP.Address.String()+":22", s.sshConfig) // TODO remove hardocoded port?
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
	contentBytes, err := io.ReadAll(f)
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
		_, _ = io.Copy(w, bytesReader)
		fmt.Fprintln(w, "\x00")
	}()

	_ = session.Run("/usr/bin/scp -t /root/") // TODO remove hardcoded remote path?
	return err
}

// WriteImageToVolume does a dd command on the remote instance via ssh
func (s *ScalewayClient) WriteImageToVolume(instanceID, deviceName string) error {
	serverResp, err := s.instanceAPI.GetServer(&instance.GetServerRequest{
		ServerID: instanceID,
	})
	if err != nil {
		return err
	}

	server := serverResp.Server

	client, err := ssh.Dial("tcp", server.PublicIP.Address.String()+":22", s.sshConfig) // TODO remove hardcoded port + use the same dial as before?
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
	log.Debugf("Shutting down server %s", instanceID)

	_, err := s.instanceAPI.ServerAction(&instance.ServerActionRequest{
		ServerID: instanceID,
		Action:   instance.ServerActionPoweroff,
	})
	if err != nil {
		return err
	}

	// code taken from scaleway-cli
	time.Sleep(10 * time.Second)

	var currentState instance.ServerState

	log.Debugf("Waiting for server to shutdown")

	for {
		serverResp, err := s.instanceAPI.GetServer(&instance.GetServerRequest{
			ServerID: instanceID,
		})
		if err != nil {
			return err
		}

		server := serverResp.Server

		if currentState != server.State {
			currentState = server.State
		}
		if server.State == instance.ServerStateStopped {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// CreateScalewayImage creates the image and delete old image and snapshot if same name
func (s *ScalewayClient) CreateScalewayImage(instanceID, volumeID, name string) error {
	oldImageID, err := s.getImageID(name, defaultScalewayCommercialType, defaultArch)
	if err == nil {
		log.Debugf("deleting image %s", oldImageID)
		err = s.instanceAPI.DeleteImage(&instance.DeleteImageRequest{
			ImageID: oldImageID,
		})
		if err != nil {
			return err
		}
	}

	oldSnapshotsResp, err := s.instanceAPI.ListSnapshots(&instance.ListSnapshotsRequest{
		Name: &name,
	}, scw.WithAllPages())
	if err == nil {
		for _, oldSnapshot := range oldSnapshotsResp.Snapshots {
			log.Debugf("deleting snapshot %s", oldSnapshot.ID)
			err = s.instanceAPI.DeleteSnapshot(&instance.DeleteSnapshotRequest{
				SnapshotID: oldSnapshot.ID,
			})
			if err != nil {
				return err
			}
		}
	}

	log.Debugf("creating snapshot %s with volume %s", name, volumeID)
	snapshotResp, err := s.instanceAPI.CreateSnapshot(&instance.CreateSnapshotRequest{
		VolumeID: volumeID,
		Name:     name,
	})
	if err != nil {
		return err
	}

	log.Debugf("creating image %s with snapshot %s", name, snapshotResp.Snapshot.ID)
	imageResp, err := s.instanceAPI.CreateImage(&instance.CreateImageRequest{
		Name:       name,
		Arch:       instance.Arch(defaultArch),
		RootVolume: snapshotResp.Snapshot.ID,
	})
	if err != nil {
		return err
	}

	log.Infof("Image %s with ID %s created", name, imageResp.Image.ID)

	return nil
}

// DeleteInstanceAndVolumes deletes the instance and the volumes attached
func (s *ScalewayClient) DeleteInstanceAndVolumes(instanceID string) error {
	serverResp, err := s.instanceAPI.GetServer(&instance.GetServerRequest{
		ServerID: instanceID,
	})
	if err != nil {
		return err
	}

	err = s.instanceAPI.DeleteServer(&instance.DeleteServerRequest{
		ServerID: instanceID,
	})
	if err != nil {
		return err
	}

	for _, volume := range serverResp.Server.Volumes {
		err = s.instanceAPI.DeleteVolume(&instance.DeleteVolumeRequest{
			VolumeID: volume.ID,
		})
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
	imageResp, err := s.instanceAPI.ListImages(&instance.ListImagesRequest{
		Name: &imageName,
	})
	if err != nil {
		return "", err
	}
	if len(imageResp.Images) != 1 {
		return "", fmt.Errorf("Image %s not found or found multiple times", imageName)
	}
	imageID := imageResp.Images[0].ID

	log.Debugf("Creating server %s on Scaleway", instanceName)
	serverResp, err := s.instanceAPI.CreateServer(&instance.CreateServerRequest{
		Name:              instanceName,
		DynamicIPRequired: &scalewayDynamicIPRequired,
		CommercialType:    instanceType,
		Image:             imageID,
		BootType:          &scalewayBootType,
	})
	if err != nil {
		return "", err
	}

	return serverResp.Server.ID, nil
}

// BootInstance boots the specified instance, and don't wait
func (s *ScalewayClient) BootInstance(instanceID string) error {
	_, err := s.instanceAPI.ServerAction(&instance.ServerActionRequest{
		ServerID: instanceID,
		Action:   instance.ServerActionPoweron,
	})
	if err != nil {
		return err
	}
	return nil
}

// ConnectSerialPort connects to the serial port of the instance
func (s *ScalewayClient) ConnectSerialPort(instanceID string) error {
	var gottyURL string
	switch s.zone {
	case scw.ZoneFrPar1:
		gottyURL = "https://tty-par1.scaleway.com/v2/"
	case scw.ZoneNlAms1:
		gottyURL = "https://tty-ams1.scaleway.com/"
	default:
		return errors.New("Instance have no region")
	}

	fullURL := fmt.Sprintf("%s?arg=%s&arg=%s", gottyURL, s.secretKey, instanceID)

	log.Debugf("Connection to %s", fullURL)
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
