package build

import (
	"fmt"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby"
)

func updateMountsAndBindsFromVolumes(image *moby.Image, m moby.Moby) (*moby.Image, error) {
	// clean image to send back
	img := *image
	if img.Mounts != nil {
		for i, mount := range *img.Mounts {
			// only care about type bind
			if mount.Type != "bind" {
				continue
			}
			// starts with / = not a volume
			if strings.HasPrefix(mount.Source, "/") {
				continue
			}
			vol := m.VolByName(mount.Source)
			if vol == nil {
				return nil, fmt.Errorf("volume %s not found in onboot image mount %d", mount.Source, i)
			}
			merged := vol.MergedDir()
			// make sure it is not read-write if the underlying volume is read-only
			if vol.ReadOnly {
				var foundReadOnly bool
				for _, opt := range mount.Options {
					if opt == "rw" {
						foundReadOnly = false
						break
					}
					if opt == "ro" {
						foundReadOnly = true
						break
					}
				}
				if !foundReadOnly {
					return nil, fmt.Errorf("volume %s is read-only, but attempting to write into container read-write", mount.Source)
				}
			}
			mount.Source = merged
		}
	}
	if img.Binds != nil {
		var newBinds []string
		for i, bind := range *img.Binds {
			parts := strings.Split(bind, ":")
			// starts with / = not a volume
			if strings.HasPrefix(parts[0], "/") {
				newBinds = append(newBinds, bind)
				continue
			}
			source := parts[0]
			// split
			vol := m.VolByName(source)
			if vol == nil {
				return nil, fmt.Errorf("volume %s not found in onboot image bin %d", source, i)
			}
			merged := vol.MergedDir()
			if vol.ReadOnly {
				if len(parts) < 3 || parts[2] != "ro" {
					return nil, fmt.Errorf("volume %s is read-only, but attempting to write into container read-write", source)
				}
			}
			parts[0] = merged
			newBinds = append(newBinds, strings.Join(parts, ":"))
		}
		img.Binds = &newBinds
	}
	if img.BindsAdd != nil {
		var newBinds []string
		for i, bind := range *img.BindsAdd {
			parts := strings.Split(bind, ":")
			// starts with / = not a volume
			if strings.HasPrefix(parts[0], "/") {
				newBinds = append(newBinds, bind)
				continue
			}
			source := parts[0]
			vol := m.VolByName(source)
			// split
			if vol == nil {
				return nil, fmt.Errorf("volume %s not found in onboot image bin %d", parts[0], i)
			}
			merged := vol.MergedDir()
			if vol.ReadOnly {
				if len(parts) < 3 || parts[2] != "ro" {
					return nil, fmt.Errorf("volume %s is read-only, but attempting to write into container read-write", source)
				}
			}
			parts[0] = merged
			newBinds = append(newBinds, strings.Join(parts, ":"))
		}
		img.BindsAdd = &newBinds
	}

	return &img, nil
}
