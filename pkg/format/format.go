package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	timeout  = 60
	ext4opts = "resize_inode,has_journal,extent,huge_file,flex_bg,uninit_bg,64bit,dir_nlink,extra_isize"
)

var (
	labelVar   string
	fsTypeVar  string
	forceVar   bool
	verboseVar bool
	drives     map[string]bool
	driveKeys  []string
)

func hasPartitions(d string) bool {
	err := exec.Command("sfdisk", "-d", d).Run()
	return err == nil
}

func isEmptyDevice(d string) (bool, error) {
	// default result
	isEmpty := false

	if verboseVar {
		log.Printf("Checking if %s is empty", d)
	}

	out, err := exec.Command("blkid", d).Output()
	if err == nil {
		log.Printf("%s has content. blkid returned: %s", d, out)
		// there is content, so exit early
		return false, nil
	}

	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			// blkid exitcode 2 (from the non-busybox version) signifies the block device has no detectable content signatures
			if status.ExitStatus() == 2 {
				if verboseVar {
					log.Printf("blkid did not find any existing content on %s.", d)
				}
				// no content detected, but continue through all checks
				isEmpty = true
			} else {
				return isEmpty, fmt.Errorf("Could not determine if %s was empty. blkid %s returned: %s (exitcode %d)", d, d, out, status.ExitStatus())
			}
		}
	}

	if hasPartitions(d) {
		log.Printf("Partition table found on device %s. Skipping.", d)
		return false, nil
	}

	// return final result
	return isEmpty, nil
}

func autoformat(label, fsType string) error {
	var first string
	for _, d := range driveKeys {
		if verboseVar {
			log.Printf("Considering auto format for device %s", d)
		}
		// break the loop with the first empty device we find
		isEmpty, err := isEmptyDevice(d)
		if err != nil {
			return err
		}
		if isEmpty == true {
			first = d
			break
		}
	}

	if first == "" {
		return fmt.Errorf("No eligible disks found")
	}

	return format(first, label, fsType, false)
}

func refreshDevicesAndWaitFor(awaitedDevice string) error {
	exec.Command("mdev", "-s").Run()

	// wait for device
	var (
		done bool
		err  error
		stat os.FileInfo
	)

	for i := 0; i < timeout; i++ {
		stat, err = os.Stat(awaitedDevice)
		if err == nil && isBlockDevice(&stat) {
			done = true
			break
		}
		time.Sleep(100 * time.Millisecond)
		exec.Command("mdev", "-s").Run()
	}
	if !done {
		var statMsg string
		if err != nil {
			statMsg = fmt.Sprintf(" - stat returned: %v", err)
		}
		return fmt.Errorf("Failed to find block device %s%s", awaitedDevice, statMsg)
	}
	// even after the device appears we still have a race
	time.Sleep(1 * time.Second)

	return nil
}

func format(d, label, fsType string, forced bool) error {
	if forced {
		// clear partitions on device if forced format and they exist
		if hasPartitions(d) {
			if verboseVar {
				log.Printf("Clearing partitions on %s because forced format was requested", d)
			}
			partCmd := exec.Command("sfdisk", "--quiet", "--delete", d)
			partCmd.Stdin = strings.NewReader(";")
			if out, err := partCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("Error deleting partitions with sfdisk: %v\n%s", err, out)
			}
		} else {
			if verboseVar {
				log.Printf("No need to clear partitions.")
			}
		}
	}

	log.Printf("Creating partition on %s", d)
	/* new disks do not have an DOS signature in sector 0
	this makes sfdisk complain. We can workaround this by letting
	fdisk create that DOS signature, by just do a "w", a write.
	http://bugs.alpinelinux.org/issues/145
	*/
	fdiskCmd := exec.Command("fdisk", d)
	fdiskCmd.Stdin = strings.NewReader("w")
	if out, err := fdiskCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Error running fdisk: %v\n%s", err, out)
	}

	// format one large partition
	partCmd := exec.Command("sfdisk", "--quiet", d)
	partCmd.Stdin = strings.NewReader(";")
	if out, err := partCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Error running sfdisk: %v\n%s", err, out)
	}

	// update status
	if err := exec.Command("blockdev", "--rereadpt", d).Run(); err != nil {
		return fmt.Errorf("Error running blockdev: %v", err)
	}

	var partition string
	// check if last char is numeric in case of nvme
	c := d[len(d)-1]
	if c > '0' && c < '9' {
		partition = fmt.Sprintf("%sp1", d)
	} else {
		partition = fmt.Sprintf("%s1", d)
	}

	if err := refreshDevicesAndWaitFor(partition); err != nil {
		return err
	}

	// mkfs
	mkfsArgs := []string{"-t", fsType}

	switch fsType {
	case "ext4":
		ext4Args := []string{"-F", "-O", ext4opts}
		if label != "" {
			ext4Args = append(ext4Args, []string{"-L", label}...)
		}
		mkfsArgs = append(mkfsArgs, ext4Args...)
	case "btrfs":
		btrfsArgs := []string{"-f"}
		if label != "" {
			btrfsArgs = append(btrfsArgs, []string{"-l", label}...)
		}
		mkfsArgs = append(mkfsArgs, btrfsArgs...)
	case "xfs":
		xfsArgs := []string{"-f"}
		if label != "" {
			xfsArgs = append(xfsArgs, []string{"-L", label}...)
		}
		mkfsArgs = append(mkfsArgs, xfsArgs...)
	default:
		log.Println("WARNING: Unsupported filesystem.")
	}

	mkfsArgs = append(mkfsArgs, partition)
	if out, err := exec.Command("mkfs", mkfsArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("Error running mkfs: %v\n%s", err, string(out))
	}

	log.Printf("Partition %s successfully created!", partition)
	return nil
}

func isBlockDevice(d *os.FileInfo) bool {
	// this probably shouldn't be so hard
	// but d.Mode()&os.ModeDevice == 0 doesn't work as expected
	mode := (*d).Sys().(*syscall.Stat_t).Mode
	return (mode & syscall.S_IFMT) == syscall.S_IFBLK
}

// return a list of all available drives
func findDrives() {
	drives = make(map[string]bool)
	driveKeys = []string{}
	ignoreExp := regexp.MustCompile(`^loop.*$|^nbd.*$|^[a-z]+[0-9]+$`)
	devs, _ := ioutil.ReadDir("/dev")
	for _, d := range devs {
		if isBlockDevice(&d) {
			if verboseVar {
				log.Printf("/dev/%s is a block device", d.Name())
			}
		} else {
			if verboseVar {
				log.Printf("/dev/%s is not a block device", d.Name())
			}
			continue
		}
		// ignore if it matches regexp
		if ignoreExp.MatchString(d.Name()) {
			if verboseVar {
				log.Printf("ignored device /dev/%s during drive autodetection", d.Name())
			}
			continue
		}
		driveKeys = append(driveKeys, filepath.Join("/dev", d.Name()))
	}
}

func init() {
	flag.BoolVar(&forceVar, "force", false, "Force format of specified single device (default false)")
	flag.StringVar(&labelVar, "label", "", "Disk label to apply")
	flag.StringVar(&fsTypeVar, "type", "ext4", "Type of filesystem to create")
	flag.BoolVar(&verboseVar, "verbose", false, "Enable verbose output (default false)")
}

func verifyBlockDevice(device string) error {
	d, err := os.Stat(device)
	if os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist", device)
	}
	if !isBlockDevice(&d) {
		return fmt.Errorf("%s is not a block device", device)
	}
	// passed checks
	return nil
}

func main() {
	flag.Parse()

	if flag.NArg() > 1 {
		log.Fatalf("Too many arguments provided")
	}

	if flag.NArg() == 0 {
		// auto-detect drives if a device to format is not explicitly specified
		findDrives()
		if err := autoformat(labelVar, fsTypeVar); err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		candidateDevice := flag.Args()[0]

		if err := verifyBlockDevice(candidateDevice); err != nil {
			log.Fatalf("%v", err)
		}

		if forceVar == true {
			if err := format(candidateDevice, labelVar, fsTypeVar, forceVar); err != nil {
				log.Fatalf("%v", err)
			}
		} else {
			// add the deviceVar to the array of devices to consider autoformatting
			driveKeys = []string{candidateDevice}
			if err := autoformat(labelVar, fsTypeVar); err != nil {
				log.Fatalf("%v", err)
			}
		}
	}
}
