package main

import (
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
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const timeout = 60

var (
	fsTypeVar   string
	stopOnError bool
	driveKeys   []string
)

// Fdisk is the JSON output from libfdisk
type Fdisk struct {
	PartitionTable struct {
		Label      string `json:"label"`
		ID         string `json:"id"`
		Device     string `json:"device"`
		Unit       string `json:"unit"`
		FirstLBA   int64  `json:"firstlba"`
		LastLBA    int64  `json:"lastlba"`
		Partitions []Partition
	} `json:"partitionTable"`
}

// Partition represents a single partition
type Partition struct {
	Node  string `json:"node"`
	Start int64  `json:"start"`
	Size  int64  `json:"size"`
	Type  string `json:"type"`
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
}

func autoextend(fsType string) error {
	for _, d := range driveKeys {
		err := exec.Command("sfdisk", "-d", d).Run()
		if err != nil {
			log.Printf("No partition table found on device %s. Skipping.", d)
			continue
		}
		if err := extend(d, fsType); err != nil {
			if stopOnError {
				return err
			}

			log.Printf("Could not extend partition on device %s. Skipping", d)
			continue
		}
	}
	return nil
}

func extend(d, fsType string) error {
	mountpoint := "/mnt/tmp"

	data, err := exec.Command("sfdisk", "-J", d).Output()
	if err != nil {
		return fmt.Errorf("Unable to get drive data for %s from sfdisk: %v", d, err)
	}

	f := Fdisk{}
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("Unable to unmarshal partition table from sfdisk: %v", err)
	}

	if len(f.PartitionTable.Partitions) > 1 {
		log.Printf("Disk %s has more than 1 partition. Skipping", d)
		return nil
	}

	partition := f.PartitionTable.Partitions[0]
	// fail on anything that isn't a Linux partition
	// 83 -> MBR/DOS Linux Partition ID
	// 0FC63DAF-8483-4772-8E79-3D69D8477DE4 -> GPT Linux Partition GUID
	if partition.Type != "83" && partition.Type != "0FC63DAF-8483-4772-8E79-3D69D8477DE4" {
		return fmt.Errorf("Partition 1 on disk %s is not a Linux Partition", d)
	}

	if f.PartitionTable.Label == "gpt" && partition.Start+partition.Size == f.PartitionTable.LastLBA {
		log.Printf("No free space on device to extend partition")
		return nil
	}
	if f.PartitionTable.Label == "dos" {
		totalSize, err := deviceSize(d)
		if err != nil {
			return fmt.Errorf("Unable to convert total size from string to int: %v", err)
		}
		if partition.Start+partition.Size == totalSize {
			log.Printf("No free space on device to extend partition")
			return nil
		}
	}

	switch fsType {
	case "ext4":
		if err := e2fsck(partition.Node, false); err != nil {
			return fmt.Errorf("Initial e2fsck failed: %v", err)
		}
		// resize2fs fails unless we set force=true here
		if err := e2fsck(partition.Node, true); err != nil {
			return fmt.Errorf("e2fsck before resize failed: %v", err)
		}

		if err := createPartition(d, partition); err != nil {
			return err
		}

		if err := exec.Command("resize2fs", partition.Node).Run(); err != nil {
			return fmt.Errorf("Error running resize2fs: %v", err)
		}

		if err := e2fsck(partition.Node, false); err != nil {
			return fmt.Errorf("e2fsck after resize failed: %v", err)
		}
	case "btrfs":
		// We don't check btrfs before or after mount as it's less susceptible to consistency errors
		// than it's extfs cousins.
		if err := os.MkdirAll(mountpoint, os.ModeDir); err != nil {
			return err
		}
		if err := createPartition(d, partition); err != nil {
			return err
		}
		if out, err := exec.Command("mount", partition.Node, mountpoint).CombinedOutput(); err != nil {
			return fmt.Errorf("Error mounting partition: %s", string(out))
		}
		if out, err := exec.Command("btrfs", "filesystem", "resize", "max", mountpoint).CombinedOutput(); err != nil {
			return fmt.Errorf("Error resizing partition: %s\n%s", err, string(out))
		}
		if out, err := exec.Command("umount", mountpoint).CombinedOutput(); err != nil {
			return fmt.Errorf("Error unmounting partition: %s", string(out))
		}
	case "xfs":
		// We don't check xfs before mounting as the xfs_check or xfs_repair utilities
		// should be used only if we suspect a file system consistency problem.
		if err := os.MkdirAll(mountpoint, os.ModeDir); err != nil {
			return err
		}
		if err := createPartition(d, partition); err != nil {
			return err
		}
		if out, err := exec.Command("mount", partition.Node, mountpoint).CombinedOutput(); err != nil {
			return fmt.Errorf("Error mounting partition: %s", string(out))
		}
		if out, err := exec.Command("xfs_growfs", mountpoint).CombinedOutput(); err != nil {
			return fmt.Errorf("Error resizing partition: %s\n%s", err, string(out))
		}
		if out, err := exec.Command("umount", mountpoint).CombinedOutput(); err != nil {
			return fmt.Errorf("Error unmounting partition: %s", string(out))
		}
		if out, err := exec.Command("xfs_repair", "-n", partition.Node).CombinedOutput(); err != nil {
			return fmt.Errorf("Error checking filesystem: %s", string(out))
		}

	default:
		return fmt.Errorf("%s is not a supported filesystem", fsType)
	}

	log.Printf("Successfully resized %s", d)
	return nil
}

func createPartition(d string, partition Partition) error {
	if err := exec.Command("sfdisk", "-q", "--delete", d).Run(); err != nil {
		return fmt.Errorf("Error erasing partition table: %v", err.Error())
	}

	createCmd := exec.Command("sfdisk", "-q", d)
	createCmd.Stdin = strings.NewReader(fmt.Sprintf("%d,,%s;", partition.Start, partition.Type))
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("Error creating partition table: %v", err)
	}

	if err := exec.Command("sfdisk", "-A", d, "1").Run(); err != nil {
		return fmt.Errorf("Error making %s bootable: %v", d, err)
	}

	// update status
	if err := rereadPartitions(d); err != nil {
		return fmt.Errorf("Error re-reading partition using ioctl: %v", err)
	}

	exec.Command("mdev", "-s").Run()

	// wait for device
	var done bool
	for i := 0; i < timeout; i++ {
		stat, err := os.Stat(partition.Node)
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
		return fmt.Errorf("Error waiting for device %s", partition.Node)
	}
	// even after the device appears we still have a race
	time.Sleep(1 * time.Second)
	return nil
}

func deviceSize(device string) (int64, error) {
	file, err := os.Open(device)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	var devsize int64
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, file.Fd(), unix.BLKGETSIZE, uintptr(unsafe.Pointer(&devsize))); errno != 0 {
		return 0, errno
	}
	return devsize, nil
}

func rereadPartitions(device string) error {
	file, err := os.Open(device)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, file.Fd(), unix.BLKRRPART, 0); errno != 0 {
		return errno
	}
	return nil
}

func e2fsck(d string, force bool) error {
	// preen
	args := []string{"-p"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, d)
	if err := exec.Command("e2fsck", args...).Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			status, ok := exiterr.Sys().(syscall.WaitStatus)
			if !ok {
				return fmt.Errorf("Unable to get status code from e2fsck")
			}
			switch status.ExitStatus() {
			case 1:
				return nil
			case 2, 3:
				return fmt.Errorf("e2fsck fixed errors but requires a reboot")
			}
		} else {
			return fmt.Errorf("Unable to cast err to ExitError")
		}
	}

	// exit code was > 4. try harder
	args[0] = "-y"
	if err := exec.Command("/sbin/e2fsck", args...).Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			status, ok := exiterr.Sys().(syscall.WaitStatus)
			if !ok {
				return fmt.Errorf("Unable to get status code from e2fsck")
			}
			switch status.ExitStatus() {
			case 1:
				return nil
			case 2, 3:
				return fmt.Errorf("e2fsck fixed errors but requires a reboot")
			default:
				return fmt.Errorf("e2fsck exited with fatal error: %v", err)
			}
		} else {
			return fmt.Errorf("Unable to cast err to ExitError")
		}
	}
	return nil
}

// return a list of all available drives
func findDrives() {
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
}

func init() {
	flag.StringVar(&fsTypeVar, "type", "ext4", "Type of filesystem to create")
	flag.BoolVar(&stopOnError, "stop-on-error", true, "Stops extending the remaining devices on first error")
}

func main() {
	flag.Parse()
	findDrives()

	if flag.NArg() == 0 {
		if err := autoextend(fsTypeVar); err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		for _, arg := range flag.Args() {
			if err := extend(arg, fsTypeVar); err != nil {
				log.Fatalf("%v", err)
			}
		}
	}
}
