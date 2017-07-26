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
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	timeout  = 60
	ext4opts = "resize_inode,has_journal,extent,huge_file,flex_bg,uninit_bg,64bit,dir_nlink,extra_isize"
)

var (
	labelVar  string
	fsTypeVar string
	drives    map[string]bool
	driveKeys []string
)

func autoformat(label, fsType string) error {
	var first string
	for _, d := range driveKeys {
		err := exec.Command("sfdisk", "-d", d).Run()
		if err == nil {
			log.Printf("Partition table found on device %s. Skipping.", d)
			continue
		}
		first = d
		break
	}

	if first == "" {
		return fmt.Errorf("No eligible disks found")
	}

	if err := format(first, label, fsType); err != nil {
		return err
	}

	return nil
}

func format(d, label, fsType string) error {
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

	exec.Command("mdev", "-s").Run()

	partition := fmt.Sprintf("%s1", d)
	// wait for device
	var done bool
	for i := 0; i < timeout; i++ {
		stat, err := os.Stat(partition)
		if err == nil {
			mode := stat.Sys().(*syscall.Stat_t).Mode
			if (mode & syscall.S_IFMT) == syscall.S_IFBLK {
				done = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
		exec.Command("mdev", "-s").Run()
	}
	if !done {
		return fmt.Errorf("Error waiting for device %s", partition)
	}
	// even after the device appears we still have a race
	time.Sleep(1 * time.Second)

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

// return a list of all available drives
func findDrives() {
	drives = make(map[string]bool)
	driveKeys = []string{}
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
	for _, d := range driveKeys {
		drives[d] = true
	}
}

func init() {
	flag.StringVar(&labelVar, "label", "", "Disk label to apply")
	flag.StringVar(&fsTypeVar, "type", "ext4", "Type of filesystem to create")
}

func main() {
	flag.Parse()

	findDrives()

	if flag.NArg() > 1 {
		log.Fatalf("Too many arguments provided")
	}

	if flag.NArg() == 0 {
		if err := autoformat(labelVar, fsTypeVar); err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		if err := format(flag.Args()[0], labelVar, fsTypeVar); err != nil {
			log.Fatalf("%v", err)
		}
	}
}
