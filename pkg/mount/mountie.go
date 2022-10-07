package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var (
	deviceVar, labelVar, uuidVar string
)

// Fdisk is the JSON output from libfdisk
type Fdisk struct {
	PartitionTable struct {
		Label      string `json:"label"`
		ID         string `json:"id"`
		Device     string `json:"device"`
		Unit       string `json:"unit"`
		FirstLBA   int    `json:"firstlba"`
		LastLBA    int    `json:"lastlba"`
		Partitions []struct {
			Node  string `json:"node"`
			Start int    `json:"start"`
			Size  int    `json:"size"`
			Type  string `json:"type"`
			UUID  string `json:"uuid"`
			Name  string `json:"name"`
		}
	} `json:"partitionTable"`
}

// mount drive/partition to mountpoint
func mount(device, mountpoint string) error {
	if out, err := exec.Command("mount", device, mountpoint).CombinedOutput(); err != nil {
		return fmt.Errorf("Error mounting %s to %s: %v\n%s", device, mountpoint, err, string(out))

	}
	return nil
}

func findDevice(pattern string) (string, error) {
	out, err := exec.Command("findfs", pattern).Output()
	if err != nil {
		return "", fmt.Errorf("Error finding device with %s: %v", pattern, err)
	}
	device := strings.TrimSpace(string(out))
	return device, nil
}

func findFirst(drives []string) (string, error) {
	var first string

	out, err := exec.Command("mount").Output()
	if err != nil {
		return "", err
	}

	mounted := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), " ")
		if _, err := os.Stat(parts[0]); os.IsNotExist(err) {
			continue
		}
		if _, ok := mounted[parts[0]]; !ok {
			mounted[parts[0]] = true
		}
	}

	for _, d := range drives {
		err := exec.Command("sfdisk", "-d", d).Run()
		if err != nil {
			log.Printf("No partition table found on device %s. Skipping.", d)
			continue
		}

		data, err := exec.Command("sfdisk", "-J", d).Output()
		if err != nil {
			log.Fatalf("Unable to get drive data for %s from sfdisk: %v", d, err)
		}

		f := Fdisk{}
		if err := json.Unmarshal(data, &f); err != nil {
			return "", fmt.Errorf("Unable to unmarshal partition table from sfdisk: %v", err)
		}

		for _, partition := range f.PartitionTable.Partitions {
			// ignore anything that isn't a Linux partition
			// 83 -> MBR/DOS Linux Partition ID
			// 0FC63DAF-8483-4772-8E79-3D69D8477DE4 -> GPT Linux Partition GUID
			if partition.Type != "83" && partition.Type != "0FC63DAF-8483-4772-8E79-3D69D8477DE4" {
				continue
			}
			if _, ok := mounted[partition.Node]; ok {
				log.Printf("%s already mounted. Skipping", partition.Node)
				continue
			}
			first = partition.Node
			break
		}
	}
	if first == "" {
		return "", fmt.Errorf("No eligible disks found")
	}
	return first, nil
}

func makeDevLinks() error {
	rex := regexp.MustCompile(`([A-Z]+)=("(?:\\.|[^"])*") ?`)

	byLabel := "/dev/disk/by-label"
	byUUID := "/dev/disk/by-uuid"
	for _, p := range []string{byLabel, byUUID} {
		err := os.MkdirAll(p, 0755)
		if err != nil {
			return err
		}
	}

	devs, err := os.ReadDir("/sys/class/block")
	if err != nil {
		return err
	}
	for _, dev := range devs {
		name := dev.Name()
		devpath := filepath.Join("/dev", name)
		outb, err := exec.Command("blkid", devpath).CombinedOutput()
		if err != nil {
			log.Printf("Unable to get blkid for %s: %v", devpath, err)
			continue
		}
		out := string(outb)
		if out == "" {
			continue
		}
		prefix := devpath + ": "
		if !strings.HasPrefix(out, prefix) {
			log.Printf("Malformed blkid for %s: %s", name, out)
			continue
		}
		out = strings.TrimPrefix(out, prefix)

		for _, match := range rex.FindAllStringSubmatch(out, -1) {
			key := match[1]

			val, err := strconv.Unquote(match[2])
			if err != nil {
				log.Printf("Failed to parse: %s\n", match[0])
				continue
			}

			switch key {
			case "LABEL":
				// This is not currently handled
				// because for compatibility we would
				// need to encode val according to
				// blkid_encode_string which hex
				// escapes certain chacters as \xXX.
				//
				// See:
				// https://github.com/systemd/systemd/blob/8d8ce9e2cd066e90c17e2d1eb1882defabb1fa63/src/udev/udev-builtin-blkid.c#L61..L66
				// https://www.kernel.org/pub/linux/utils/util-linux/v2.21/libblkid-docs/libblkid-Encoding-utils.html
			case "UUID":
				// Strictly the value should be
				// encoded here as with "LABEL" but we
				// take the chance that a string UUID
				// is unlikely to contain any unsafe
				// characters.
				sympath := filepath.Join(byUUID, val)
				// udev makes these relative links, copy that behaviour.
				tgtpath := filepath.Join("..", "..", name)
				if err := os.Symlink(tgtpath, sympath); err != nil {
					log.Printf("Failed to create %q: %v", sympath, err)
					continue
				}
			case "TYPE":
				// uninteresting
			default:
				log.Printf("unused %q blkid property %q in %q", name, key, match[0])
			}
		}
	}
	return nil
}

// return a list of all available drives
func findDrives() []string {
	var driveKeys []string
	ignoreExp := regexp.MustCompile(`^loop.*$|^nbd.*$|^[a-z]+[0-9]+$`)
	devs, _ := ioutil.ReadDir("/dev")
	for _, d := range devs {
		// this probably shouldn't be so hard
		// but d.Mode()&os.ModeDevice == 0 doesn't work as expected
		mode := d.Sys().(*syscall.Stat_t).Mode
		if (mode & syscall.S_IFMT) != syscall.S_IFBLK {
			continue
		}
		// ignore if it matches regexp
		if ignoreExp.MatchString(d.Name()) {
			continue
		}
		driveKeys = append(driveKeys, filepath.Join("/dev", d.Name()))
	}
	sort.Strings(driveKeys)
	return driveKeys
}

func init() {
	flag.StringVar(&deviceVar, "device", "", "Name of the device to mount")
	flag.StringVar(&labelVar, "label", "", "Label of the device to mount")
	flag.StringVar(&uuidVar, "uuid", "", "UUID of the device to mount")
}

func main() {
	flag.Parse()

	var mountpoint string

	switch flag.NArg() {
	case 0:
		log.Fatal("No mountpoints provided")
	case 1:
		mountpoint = flag.Args()[0]
	case 2:
		deviceVar = flag.Args()[0]
		mountpoint = flag.Args()[1]
	default:
		log.Fatalf("Too many arguments")
	}

	err := os.MkdirAll(mountpoint, 0755)
	if err != nil {
		log.Fatalf("Unable to create mountpoint %s: %v", mountpoint, err)
	}
	if deviceVar == "" && labelVar != "" {
		deviceVar, err = findDevice(fmt.Sprintf("LABEL=%s", labelVar))
		if err != nil {
			log.Fatal(err)
		}
	}
	if deviceVar == "" && uuidVar != "" {
		deviceVar, err = findDevice(fmt.Sprintf("UUID=%s", uuidVar))
		if err != nil {
			log.Fatal(err)
		}
	}

	if deviceVar == "" {
		// find first device
		drives := findDrives()
		first, err := findFirst(drives)
		if err != nil {
			log.Fatal(err)
		}
		deviceVar = first
	}

	if err := mount(deviceVar, mountpoint); err != nil {
		log.Fatal(err)
	}

	if err := makeDevLinks(); err != nil {
		log.Printf("Failed to make /dev/ links for: %v", err)
	}

}
