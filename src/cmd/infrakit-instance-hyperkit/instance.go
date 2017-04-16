package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/hyperkit/go"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	"github.com/rneugeba/iso9660wrap"
)

// NewHyperKitPlugin creates an instance plugin for hyperkit.
func NewHyperKitPlugin(vmDir, hyperkit, vpnkitSock string) instance.Plugin {
	return &hyperkitPlugin{VMDir: vmDir,
		HyperKit:   hyperkit,
		VPNKitSock: vpnkitSock,
		DiskDir:    path.Join(vmDir, "disks"),
	}
}

type hyperkitPlugin struct {
	// VMDir is the path to a directory where per VM state is kept
	VMDir string

	// Hyperkit is the path to the hyperkit executable
	HyperKit string

	// VPNKitSock is the path to the VPNKit Unix domain socket.
	VPNKitSock string

	// DiskDir is the path to persistent (across reboots) disk images
	DiskDir string
}

// Validate performs local validation on a provision request.
func (p hyperkitPlugin) Validate(req *types.Any) error {
	return nil
}

// Provision creates a new instance.
func (p hyperkitPlugin) Provision(spec instance.Spec) (*instance.ID, error) {

	var properties map[string]interface{}

	if spec.Properties != nil {
		if err := spec.Properties.Decode(&properties); err != nil {
			return nil, fmt.Errorf("Invalid instance properties: %s", err)
		}
	}

	if properties["kernel+initrd"] == nil {
		return nil, errors.New("Property 'kernel+initrd' must be set")
	}
	if properties["CPUs"] == nil {
		properties["CPUs"] = 1
	}
	if properties["Memory"] == nil {
		properties["Memory"] = 512
	}
	diskSize := 0
	if properties["Disk"] != nil {
		diskSize = int(properties["Disk"].(float64))
	}

	instanceDir, err := ioutil.TempDir(p.VMDir, "infrakit-")
	if err != nil {
		return nil, err
	}
	id := instance.ID(path.Base(instanceDir))
	log.Infof("[%s] New instance", id)

	logicalID := string(id)
	vpnkitKeyStr := ""

	diskImage := ""
	if spec.LogicalID != nil {
		logicalID = string(*spec.LogicalID)
		// The LogicalID may be a IP address. If so, translate
		// it into a magic UUID which cause VPNKit to assign a
		// fixed IP address
		if ip := net.ParseIP(logicalID); len(ip) > 0 {
			vpnkitkey := make([]byte, 16)
			vpnkitkey[12] = ip.To4()[0]
			vpnkitkey[13] = ip.To4()[1]
			vpnkitkey[14] = ip.To4()[2]
			vpnkitkey[15] = ip.To4()[3]
			vpnkitKeyStr = fmt.Sprintf("%x-%x-%x-%x-%x", vpnkitkey[0:4], vpnkitkey[4:6], vpnkitkey[6:8], vpnkitkey[8:10], vpnkitkey[10:])
		}
		// If a LogicalID is supplied and the Disk size is
		// non-zero, we place the disk in a special directory
		// so it persists across reboots.
		if diskSize != 0 {
			diskImage = path.Join(p.DiskDir, logicalID+".img")
		}
	}

	isoImage := ""
	if spec.Init != "" {
		isoImage = path.Join(instanceDir, "data.iso")
		outfh, err := os.OpenFile(isoImage, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Cannot create user data ISO: %s", err)
		}
		err = iso9660wrap.WriteBuffer(outfh, []byte(spec.Init), "config")
		if err != nil {
			log.Fatalf("Cannot write user data ISO: %s", err)
		}
		outfh.Close()
	}

	log.Infof("[%s] LogicalID: %s", id, logicalID)
	log.Debugf("[%s] VPNKitKey: %s", id, vpnkitKeyStr)

	// Start a HyperKit instance
	h, err := hyperkit.New(p.HyperKit, p.VPNKitSock, instanceDir)
	if err != nil {
		return nil, err
	}
	h.Kernel = properties["kernel+initrd"].(string) + "-bzImage"
	h.Initrd = properties["kernel+initrd"].(string) + "-initrd.img"
	h.VPNKitKey = vpnkitKeyStr
	h.DiskImage = diskImage
	h.ISOImage = isoImage
	h.CPUs = int(properties["CPUs"].(float64))
	h.Memory = int(properties["Memory"].(float64))
	h.DiskSize = diskSize
	h.Console = hyperkit.ConsoleFile
	log.Infof("[%s] Booting: %s/%s", id, h.Kernel, h.Initrd)
	log.Infof("[%s] %d CPUs, %dMB Memory, %dMB Disk (%s)", id, h.CPUs, h.Memory, h.DiskSize, h.DiskImage)

	err = h.Start("console=ttyS0")
	if err != nil {
		return nil, err
	}
	log.Infof("[%s] Started", id)

	if err := ioutil.WriteFile(path.Join(instanceDir, "logical.id"), []byte(logicalID), 0644); err != nil {
		return nil, err
	}

	tagData, err := types.AnyValue(spec.Tags)
	if err != nil {
		return nil, err
	}

	log.Debugf("[%s] tags: %s", id, tagData)
	if err := ioutil.WriteFile(path.Join(instanceDir, "tags"), tagData.Bytes(), 0644); err != nil {
		return nil, err
	}

	return &id, nil
}

// Label labels the instance
func (p hyperkitPlugin) Label(instance instance.ID, labels map[string]string) error {
	instanceDir := path.Join(p.VMDir, string(instance))
	tagFile := path.Join(instanceDir, "tags")
	buff, err := ioutil.ReadFile(tagFile)
	if err != nil {
		return err
	}

	tags := map[string]string{}
	err = types.AnyBytes(buff).Decode(&tags)
	if err != nil {
		return err
	}

	for k, v := range labels {
		tags[k] = v
	}

	encoded, err := types.AnyValue(tags)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(tagFile, encoded.Bytes(), 0644)
}

// Destroy terminates an existing instance.
func (p hyperkitPlugin) Destroy(id instance.ID) error {
	log.Info("Destroying VM: ", id)

	instanceDir := path.Join(p.VMDir, string(id))
	_, err := os.Stat(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("Instance does not exist")
		}
	}

	h, err := hyperkit.FromState(instanceDir)
	if err != nil {
		return err
	}
	err = h.Stop()
	if err != nil {
		return err
	}
	err = h.Remove(false)
	if err != nil {
		return err
	}
	return nil
}

// DescribeInstances returns descriptions of all instances matching all of the provided tags.
func (p hyperkitPlugin) DescribeInstances(tags map[string]string, properties bool) ([]instance.Description, error) {
	files, err := ioutil.ReadDir(p.VMDir)
	if err != nil {
		return nil, err
	}

	descriptions := []instance.Description{}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		instanceDir := path.Join(p.VMDir, file.Name())

		tagData, err := ioutil.ReadFile(path.Join(instanceDir, "tags"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		instanceTags := map[string]string{}
		if err := types.AnyBytes(tagData).Decode(&instanceTags); err != nil {
			return nil, err
		}

		allMatched := true
		for k, v := range tags {
			value, exists := instanceTags[k]
			if !exists || v != value {
				allMatched = false
				break
			}
		}

		if allMatched {
			var logicalID *instance.LogicalID
			id := instance.ID(file.Name())

			h, err := hyperkit.FromState(instanceDir)
			if err != nil {
				log.Warningln("Could not get instance data. Id: ", id)
				p.Destroy(id)
				continue
			}
			if !h.IsRunning() {
				log.Warningln("Instance is not running. Id: ", id)
				p.Destroy(id)
				continue
			}

			lidData, err := ioutil.ReadFile(path.Join(instanceDir, "logical.id"))
			if err != nil {
				log.Warningln("Could not get logical ID. Id: ", id)
				p.Destroy(id)
				continue
			}
			lid := instance.LogicalID(lidData)
			logicalID = &lid

			descriptions = append(descriptions, instance.Description{
				ID:        id,
				LogicalID: logicalID,
				Tags:      instanceTags,
			})
		}
	}

	return descriptions, nil
}
