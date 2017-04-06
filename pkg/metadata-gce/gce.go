package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"syscall"
	"time"
)

const (
	project  = "http://metadata.google.internal/computeMetadata/v1/project/"
	instance = "http://metadata.google.internal/computeMetadata/v1/instance/"
)

// If optional not set, will panic. Optional will allow 404
// We assume most failure cases are that this code is included in a non Google Cloud
// environment, which should generally be ok, so just fail fast.
func metadata(url string, optional bool) []byte {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		log.Fatalf("http NewRequest failed: %v", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		// Probably not running on Google Cloud but this package included
		log.Fatalf("Could not contact Google Cloud Metadata service: %v", err)
	}
	if optional && resp.StatusCode == 404 {
		return []byte{}
	}
	if resp.StatusCode != 200 {
		// Probably not running on Google Cloud but this package included
		log.Fatalf("Google Cloud Metadata Server http error: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read http response: %v", err)
	}
	return body
}

func main() {
	hostname := metadata(instance+"hostname", false)
	err := syscall.Sethostname(hostname)
	if err != nil {
		log.Printf("Failed to set hostname: %v", err)
	}
	sshKeys := metadata(project+"attributes/sshKeys", true)
	// TODO also retrieve the instance keys and respect block project keys see https://cloud.google.com/compute/docs/instances/ssh-keys

	// the keys have usernames attached, but as a simplification we are going to add them all to one root file
	// TODO split them into individual user files and make the ssh container construct those users

	rootKeys := ""
	for _, line := range strings.Split(string(sshKeys), "\n") {
		parts := strings.SplitN(line, ":", 2)
		// ignoring username for now
		if len(parts) == 2 {
			rootKeys = rootKeys + parts[1] + "\n"
		}
	}

	err = ioutil.WriteFile("/etc/ssh/authorized_keys", []byte(rootKeys), 0600)
	if err != nil {
		log.Printf("Failed to write ssh keys: %v", err)
	}
}
