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

	// Hostname is the filename in configPath where the hostname is stored
	Hostname = "hostname"

	// SSH is the path where sshd configuration from the provider is stored
	SSH = "ssh"
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

func main() {
	providers := []string{"aws", "gcp", "vultr", "packet", "cdrom"}
	if len(os.Args) > 1 {
		providers = os.Args[1:]
	}
	for _, p := range providers {
		switch p {
		case "aws":
			netProviders = append(netProviders, NewAWS())
		case "gcp":
			netProviders = append(netProviders, NewGCP())
		case "packet":
			netProviders = append(netProviders, NewPacket())
		case "vultr":
			netProviders = append(netProviders, NewVultr())
		case "cdrom":
			cdromProviders = ListCDROMs()
		default:
			log.Fatalf("Unrecognised metadata provider: %s", p)
		}
	}

	if err := os.MkdirAll(ConfigPath, 0755); err != nil {
		log.Fatalf("Could not create %s: %s", ConfigPath, err)
	}

	var p Provider
	var userdata []byte
	var err error
	found := false
	for _, p = range netProviders {
		if p.Probe() {
			log.Printf("%s: Probe succeeded", p)
			userdata, err = p.Extract()
			found = true
			break
		}
	}
	if !found {
		for _, p = range cdromProviders {
			log.Printf("Trying %s", p.String())
			if p.Probe() {
				log.Printf("%s: Probe succeeded", p)
				userdata, err = p.Extract()
				found = true
				break
			}
		}
	}

	if !found {
		log.Printf("No metadata/userdata found. Bye")
		return
	}

	if err != nil {
		log.Printf("Error during metadata probe: %s", err)
	}

	err = ioutil.WriteFile(path.Join(ConfigPath, "provider"), []byte(p.String()), 0644)
	if err != nil {
		log.Printf("Error writing metadata provider: %s", err)
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
	cm, ok := fd.(map[string]interface{})
	if !ok {
		log.Printf("Could convert JSON to desired format: %s", fd)
		return nil
	}
	for d, val := range cm {
		dir := path.Join(ConfigPath, d)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create %s: %s", dir, err)
			continue
		}
		files, ok := val.(map[string]interface{})
		if !ok {
			log.Printf("Could convert JSON for files: %s", val)
			continue
		}
		for f, i := range files {
			p := uint64(0644)
			var c string

			switch fi := i.(type) {
			case map[string]interface{}:
				if _, ok := fi["perm"]; !ok {
					log.Printf("No permission provided %s", f)
					continue
				}
				if _, ok := fi["content"]; !ok {
					log.Printf("No content provided %s", f)
					continue
				}
				c = fi["content"].(string)
				if p, err = strconv.ParseUint(fi["perm"].(string), 8, 32); err != nil {
					log.Printf("Failed to parse permission %s: %s", fi, err)
					continue
				}
			case string:
				c = fi
			default:
				log.Printf("Couldn't convert JSON for items: %s", i)
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
