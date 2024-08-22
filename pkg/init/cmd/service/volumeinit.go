package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func volumeInitCmd(ctx context.Context) int {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet("volume", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s volume\n\n", invoked)
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}

	path := flags.String("path", defaultVolumesPath, "Path to volume configs")

	// Set up volumes
	vols, err := os.ReadDir(*path)
	// just skip if there is an error, eg no such path
	if err != nil {
		return 1
	}
	// go through each volume, ensure that the volPath/merged exists as a directory,
	// and is one of:
	// - read-only: i.e. no tmp exists, merged bindmounted to lower
	// - read-write: i.e. tmp exists, overlayfs lower/upper/merged
	for _, vol := range vols {
		subs, err := os.ReadDir(filepath.Join(*path, vol.Name()))
		if err != nil {
			log.WithError(err).Errorf("Error reading volume %s", vol.Name())
			return 1
		}
		var hasLower, hasMerged, readWrite bool
		for _, sub := range subs {
			switch sub.Name() {
			case "lower":
				hasLower = true
			case "tmp":
				readWrite = true
			case "merged":
				hasMerged = true
			}
		}
		if !hasMerged {
			log.Errorf("Volume %s does not have a merged directory", vol.Name())
			return 1
		}
		if !hasLower {
			log.Errorf("Volume %s does not have a lower directory", vol.Name())
			return 1
		}
		lowerDir := filepath.Join(*path, vol.Name(), "lower")
		mergedDir := filepath.Join(*path, vol.Name(), "merged")
		// need a tmpfs to create the workdir and upper
		tmpDir := filepath.Join(*path, vol.Name(), "tmp")
		if err := unix.Mount("tmpfs", tmpDir, "tmpfs", unix.MS_RELATIME, ""); err != nil {
			log.WithError(err).Errorf("Error creating tmpDir for volume %s", vol.Name())
			return 1
		}
		workDir := filepath.Join(tmpDir, "work")
		upperDir := filepath.Join(tmpDir, "upper")
		if err := os.Mkdir(upperDir, 0755); err != nil {
			log.WithError(err).Errorf("Error creating upper dir for volume %s", vol.Name())
			return 1
		}
		if err := os.Mkdir(workDir, 0755); err != nil {
			log.WithError(err).Errorf("Error creating work dir for volume %s", vol.Name())
			return 1
		}
		// and let's mount the actual dir
		mountOps := []string{fmt.Sprintf("lowerdir=%s", lowerDir), fmt.Sprintf("upperdir=%s", upperDir), fmt.Sprintf("workdir=%s", workDir)}

		if !readWrite {
			log.Infof("Volume %s is read-only", vol.Name())
			mountOps = append(mountOps, "ro")
		} else {
			log.Infof("Volume %s is read-write", vol.Name())
		}
		data := strings.Join(mountOps, ",")
		if err := unix.Mount("overlay", mergedDir, "overlay", unix.MS_RELATIME, data); err != nil {
			log.WithError(err).Errorf("Error overlay-mounting volume %s", vol.Name())
			return 1
		}
	}
	return 0
}
