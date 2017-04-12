package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"syscall"
)

const (
	// ConfigPath is where the data is extracted to
	ConfigPath = "/var/config"

	// MountPoint is where the CDROM is mounted
	MountPoint = "/cdrom"

	// Hostname is the filename in configPath where the hostname is stored
	Hostname = "hostname"

	// SSH is the path where sshd configuration from the provider is stored
	SSH = "ssh"

	// TODO(rneugeba): Need to check this is the same everywhere
	cdromDev = "/dev/sr0"
)

// Provider is a generic interface for metadata/userdata providers.
type Provider interface {
	// String should return a unique name for the Provider
	String() string

	// Probe returns true if the provider was detected.
	Probe() bool

	// Extract user data. This may write some data, specific to a
	// provider, to ConfigPath and should return the generic userdata.
	Extract() ([]byte, error)
}

// netProviders is a list of Providers offering metadata/userdata over the network
var netProviders []Provider

// cdromProviders is a list of Providers offering metadata/userdata data via CDROM
var cdromProviders []Provider

func init() {
	netProviders = []Provider{NewGCP()}
	cdromProviders = []Provider{NewCDROM()}
}

func main() {
	if err := os.MkdirAll(ConfigPath, 0755); err != nil {
		log.Fatalf("Could not create %s: %s", ConfigPath, err)
	}

	var userdata []byte
	var err error
	found := false
	for _, p := range netProviders {
		if p.Probe() {
			log.Printf("%s: Probe succeeded", p)
			userdata, err = p.Extract()
			found = true
			break
		}
	}
	if !found {
		log.Printf("Trying CDROM")
		if err := os.Mkdir(MountPoint, 0755); err != nil {
			log.Printf("CDROM: Failed to create %s: %s", MountPoint, err)
			goto ErrorOut
		}
		if err := mountCDROM(cdromDev, MountPoint); err != nil {
			log.Printf("Failed to mount cdrom: %s", err)
			goto ErrorOut
		}
		defer syscall.Unmount(MountPoint, 0)
		// Don't worry about removing MountPoint. We are in a container

		for _, p := range cdromProviders {
			if p.Probe() {
				log.Printf("%s: Probe succeeded", p)
				userdata, err = p.Extract()
				found = true
				break
			}
		}
	}

ErrorOut:
	if !found {
		log.Printf("No metadata/userdata found. Bye")
		return
	}

	if err != nil {
		log.Printf("Error during metadata probe: %s", err)
	}

	if userdata != nil {
		if err := processUserData(userdata); err != nil {
			log.Printf("Could not extract user data: %s", err)
		}
	}

	// Handle setting the hostname as a special case. We want to
	// do this early and don't really want another container for it.
	hostname, err := ioutil.ReadFile(path.Join(ConfigPath, Hostname))
	if err == nil {
		err := syscall.Sethostname(hostname)
		if err != nil {
			log.Printf("Failed to set hostname: %s", err)
		} else {
			log.Printf("Set hostname to: %s", string(hostname))
		}
	}
}

// If the userdata is a json file, create a directory/file hierarchy.
// Example:
// {
//    "foobar" : {
//        "foo" : {
//            "perm": "0644",
//            "content": "hello"
//        }
// }
// Will create foobar/foo with mode 0644 and content "hello"
func processUserData(data []byte) error {
	// Always write the raw data to a file
	err := ioutil.WriteFile(path.Join(ConfigPath, "userdata"), data, 0644)
	if err != nil {
		log.Printf("Could not write userdata: %s", err)
		return err
	}

	var fd interface{}
	if err := json.Unmarshal(data, &fd); err != nil {
		// Userdata is no JSON, presumably...
		log.Printf("Could not unmarshall userdata: %s", err)
		// This is not an error
		return nil
	}
	cm := fd.(map[string]interface{})
	for d, val := range cm {
		dir := path.Join(ConfigPath, d)
		if err := os.Mkdir(dir, 0755); err != nil {
			log.Printf("Failed to create %s: %s", dir, err)
			continue
		}
		files := val.(map[string]interface{})
		for f, i := range files {
			fi := i.(map[string]interface{})
			if _, ok := fi["perm"]; !ok {
				log.Printf("No permission provided %s:%s", f, fi)
				continue
			}
			if _, ok := fi["content"]; !ok {
				log.Printf("No content provided %s:%s", f, fi)
				continue
			}
			c := fi["content"].(string)
			p, err := strconv.ParseUint(fi["perm"].(string), 8, 32)
			if err != nil {
				log.Printf("Failed to parse permission %s: %s", fi, err)
				continue
			}
			if err := ioutil.WriteFile(path.Join(dir, f), []byte(c), os.FileMode(p)); err != nil {
				log.Printf("Failed to write %s/%s: %s", dir, f, err)
				continue

			}
		}
	}

	return nil
}

// mountCDROM mounts a CDROM/DVD device under mountPoint
func mountCDROM(device, mountPoint string) error {
	// We may need to poll a little for device ready
	return syscall.Mount(device, mountPoint, "iso9660", syscall.MS_RDONLY, "")
}
