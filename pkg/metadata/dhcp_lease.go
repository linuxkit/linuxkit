package main

import (
	"path/filepath"
	"regexp"

	"github.com/thoas/go-funk"
)

// DHCPServerLeaseFileLocator Find dhcp server by looking up the lease file generated.
// This implies the existence of dhclient
type DHCPServerLeaseFileLocator struct {
}

const (
	// DHCPDirsPattern This is to match dhcp and dhcp3 and various many others
	DHCPDirsPattern = "/var/lib/dhcp*/*.leases"
)

// DHCPServerRegex I do not care about whether the IP address is correct, rather I assumed the DHCP client
// will always produce and validate the DHCP server to be in octet range
// IP address example: 127.10.0.1
var DHCPServerRegex = regexp.MustCompile(`option[ \t]+dhcp-server-identifier[ \t]+((?:[0-9]{1,3}\.){3}[0-9]{1,3});?`)

// Probe Guesses possible DHCP server addresses
// Used here to figure out the metadata service server
func (d *DHCPServerLeaseFileLocator) Probe() (possibleAddresses []string, err error) {
	// i actually wanted set data structure in golang...and golang team said barely anybody use it so nope
	// f*ck it, i roll my own. ken thompson why would you make such a stupid mistake?
	defer func() {
		if err == nil {
			possibleAddresses = funk.UniqString(possibleAddresses)
		}
	}()

	var matches []string
	// error or no match
	if matches, err = filepath.Glob(DHCPDirsPattern); err != nil || len(matches) < 1 {
		return
	}

	for _, value := range matches {
		// fail fast if there's an error already
		if err != nil {
			return
		}
		func() {
			var file MemoryMappedFile
			if file, err = OpenMemoryMappedFile(value); err != nil {
				return
			}
			defer func() {
				if err = file.Close(); err != nil {
					return
				}
			}()

			// submatch 0 is the matched expression source
			// so starting from submatch 1 it will be the capture groups...at least 2 elements
			if submatches := DHCPServerRegex.FindSubmatch(*file.data); len(submatches) > 1 {
				possibleAddresses = funk.Map(funk.Tail(submatches), func(submatch []byte) string {
					return string(submatch)
				}).([]string)
			}
		}()
	}
	return
}
