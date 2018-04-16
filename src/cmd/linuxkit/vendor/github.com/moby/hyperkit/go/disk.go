package hyperkit

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	mib = int64(1024 * 1024)
)

/*-------.
| Disk.  |
`-------*/

// Disk in an interface for qcow2 and raw disk images.
type Disk interface {
	// GetPath returns the location of the disk image file.
	GetPath() string
	// SetPath changes the location of the disk image file.
	SetPath(p string)
	// GetSize returns the desired disk size.
	GetSize() int
	// GetCurrentSize returns the current disk size in MiB.
	GetCurrentSize() (int, error)
	// String returns the path.
	String() string

	// Exists iff the disk image file can be stat'd without error.
	Exists() bool
	// Ensure creates the disk image if needed, and resizes it if needed.
	Ensure() error
	// Stop can be called when hyperkit has quit.  It performs sanity checks, compaction, etc.
	Stop() error

	// AsArgument returns the command-line option to pass after `-s <slot>:0,` to hyperkit for this disk.
	AsArgument() string

	create() error
	resize() error
}

// DiskFormat describes the physical format of the disk data
type DiskFormat int

const (
	// DiskFormatQcow means the disk is a qcow2.
	DiskFormatQcow DiskFormat = iota

	// DiskFormatRaw means the disk is a raw file.
	DiskFormatRaw
)

// GetDiskFormat computes the format based on the path's extensions.
func GetDiskFormat(path string) DiskFormat {
	switch ext := filepath.Ext(path); ext {
	case ".qcow2":
		return DiskFormatQcow
	case ".raw", ".img":
		return DiskFormatRaw
	default:
		log.Debugf("hyperkit: Unknown disk extension %q, will use raw format", path)
		return DiskFormatRaw
	}
}

// NewDisk creates a qcow/raw disk configuration based on the spec.
func NewDisk(spec string, size int) (Disk, error) {
	u, err := url.Parse(spec)
	if err != nil {
		return nil, fmt.Errorf("invalid disk path %q: %v", spec, err)
	}
	switch path := u.Path; GetDiskFormat(path) {
	case DiskFormatRaw:
		return &RawDisk{
			Path: path,
			Size: size,
			Trim: true,
		}, nil
	case DiskFormatQcow:
		return &QcowDisk{
			Path: path,
			Size: size,
		}, nil
	}
	return nil, fmt.Errorf("impossible")
}

// exists iff the image file can be stat'd without error.
func exists(d Disk) bool {
	_, err := os.Stat(d.GetPath())
	if err != nil && !os.IsNotExist(err) {
		log.Debugf("hyperkit: cannot stat %q: %v", d, err)
	}
	return err == nil
}

// ensure creates the disk image if needed, and resizes it if needed.
func ensure(d Disk) error {
	current, err := d.GetCurrentSize()
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return d.create()
	}
	if current < d.GetSize() {
		return d.resize()
	}
	if d.GetSize() < current {
		log.Errorf("hyperkit: Cannot safely shrink %q from %dMiB to %dMiB", d, current, d.GetSize())
	}
	return nil
}

// diskDriver to use.
//
// Dave Scott writes:
//
// > Regarding TRIM and raw disks
// > (https://github.com/docker/pinata/pull/8235/commits/0e2c7c2e21114b4ed61589bd42b720f7d88c0d8e):
// > it works like this: the `ahci-hd` virtual hardware in hyperkit
// > exposes the `ATA_SUPPORT_DSM_TRIM` capability
// > (https://github.com/moby/hyperkit/blob/81fa6279fcb17e8435f3cec0978e9aa3af02e63b/src/lib/pci_ahci.c#L996)
// > if the `fcntl(F_PUNCHHOLE)`
// > (https://github.com/moby/hyperkit/blob/81fa6279fcb17e8435f3cec0978e9aa3af02e63b/src/lib/block_if.c#L276)
// > API works on the raw file (it's dynamically detected so on HFS+ it's
// > disabled and on APFS it's enabled) -> TRIM on raw doesn't need any
// > special flags set in the Go code; the special flags are only for the
// > TRIM on qcow implementation. When images are deleted in the VM the
// > `trim-after-delete`
// > (https://github.com/linuxkit/linuxkit/tree/master/pkg/trim-after-delete)
// > daemon calls `fstrim /var/lib/docker` which causes Linux to emit the
// > TRIM commands to hyperkit, which calls `fcntl`, which tells macOS to
// > free the space in the file, visible in `ls -sl`.
// >
// > Unfortunately the `virtio-blk` protocol doesn't support `TRIM`
// > requests at all so we have to use `ahci-hd` (if you try to run
// > `fstrim /var/lib/docker` with `virtio-blk` it'll give an `ioctl`
// > error).
func diskDriver(trim bool) string {
	if trim {
		return "ahci-hd"
	}
	return "virtio-blk"
}

/*----------.
| RawDisk.  |
`----------*/

// RawDisk describes a raw disk image file.
type RawDisk struct {
	// Path specifies where the image file will be.
	Path string `json:"path"`
	// Size specifies the size of the disk.
	Size int `json:"size"`
	// Format is passed as-is to the driver.
	Format string `json:"format"`
	// Trim specifies whether we should trim the image file.
	Trim bool `json:"trim"`
}

// GetPath returns the location of the disk image file.
func (d *RawDisk) GetPath() string {
	return d.Path
}

// SetPath changes the location of the disk image file.
func (d *RawDisk) SetPath(p string) {
	d.Path = p
}

// GetSize returns the desired disk size.
func (d *RawDisk) GetSize() int {
	return d.Size
}

// String returns the path.
func (d *RawDisk) String() string {
	return d.Path
}

// Exists iff the image file can be stat's without error.
func (d *RawDisk) Exists() bool {
	return exists(d)
}

// Ensure creates the disk image if needed, and resizes it if needed.
func (d *RawDisk) Ensure() error {
	return ensure(d)
}

// Create a disk.
func (d *RawDisk) create() error {
	log.Infof("hyperkit: Create %q", d)
	f, err := os.Create(d.Path)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return d.resize()
}

// GetCurrentSize returns the current disk size in MiB.
func (d *RawDisk) GetCurrentSize() (int, error) {
	fileinfo, err := os.Stat(d.Path)
	if err != nil {
		return 0, err
	}
	return int(fileinfo.Size() / mib), nil
}

// Resize the virtual size of the disk
func (d *RawDisk) resize() error {
	s, err := d.GetCurrentSize()
	if err != nil {
		return fmt.Errorf("Cannot resize %q: %v", d, err)
	}
	log.Infof("hyperkit: Resize %q from %vMiB to %vMiB", d, s, d.GetSize())
	// APFS exhibits a weird behavior wrt sparse files: we cannot
	// create (or grow) them "too fast": there's a limit,
	// apparently related to the available disk space.  However,
	// if the additional space is small enough, we can procede way
	// beyond the available disk space.  So grow incrementally,
	// by steps of 1GB.
	for s < d.Size {
		s += 1000
		if d.Size < s {
			s = d.Size
		}
		if err := os.Truncate(d.Path, int64(s)*mib); err != nil {
			return fmt.Errorf("Cannot resize %q to %vMiB: %v", d, s, err)
		}
	}
	log.Infof("hyperkit: Resized %q to %vMiB", d, d.GetSize())
	return nil
}

// Stop cleans up this disk when we are quitting.
func (d *RawDisk) Stop() error {
	return nil
}

// AsArgument returns the command-line option to pass after `-s <slot>:0,` to hyperkit for this disk.
func (d *RawDisk) AsArgument() string {
	res := fmt.Sprintf("%s,%s", diskDriver(d.Trim), d.Path)
	if d.Format != "" {
		res += ",format=" + d.Format
	}
	return res
}

/*-----------.
| QcowDisk.  |
`-----------*/

// QcowDisk describes a qcow2 disk image file.
type QcowDisk struct {
	// Path specifies where the image file will be.
	Path string `json:"path"`
	// Size specifies the size of the disk.
	Size int `json:"size"`
	// Format is passed as-is to the driver.
	Format string `json:"format"`
	// Trim specifies whether we should trim the image file.
	Trim bool `json:"trim"`
	// QcowToolPath is the path to the binary to use to manage this image.
	// Defaults to "qcow-tool" when empty.
	QcowToolPath   string
	OnFlush        string
	CompactAfter   int
	KeepErased     int
	RuntimeAsserts bool
	Stats          string
}

// GetPath returns the location of the disk image file.
func (d *QcowDisk) GetPath() string {
	return d.Path
}

// SetPath changes the location of the disk image file.
func (d *QcowDisk) SetPath(p string) {
	d.Path = p
}

// GetSize returns the desired disk size.
func (d *QcowDisk) GetSize() int {
	return d.Size
}

// String returns the path.
func (d *QcowDisk) String() string {
	return d.Path
}

// QcowTool prepares a call to qcow-tool on this image.
func (d *QcowDisk) QcowTool(verb string, args ...string) *exec.Cmd {
	if d.QcowToolPath == "" {
		d.QcowToolPath = "qcow-tool"
	}
	return exec.Command(d.QcowToolPath, append([]string{verb, d.Path}, args...)...)
}

func run(cmd *exec.Cmd) (string, error) {
	buf, err := cmd.CombinedOutput()
	out := string(buf)
	log.Debugf("hyperkit: ran %v: out=%q, err=%v", cmd.Args, out, err)
	return out, err
}

// Exists iff the image file can be stat'd without error.
func (d *QcowDisk) Exists() bool {
	return exists(d)
}

// Ensure creates the disk image if needed, and resizes it if needed.
func (d *QcowDisk) Ensure() error {
	if d.Trim {
		log.Infof("hyperkit: %v: TRIM is enabled; recycling thread will keep %v sectors free and will compact after %v more sectors are free",
			d, d.KeepErased, d.CompactAfter)
	}
	if d.RuntimeAsserts {
		log.Warnf("hyperkit: %v: Expensive runtime checks are enabled", d)
	}
	return ensure(d)
}

// Create a disk with the given size in MiB
func (d *QcowDisk) create() error {
	log.Infof("hyperkit: Create %q", d)
	_, err := run(d.QcowTool("create", "--size", fmt.Sprintf("%dMiB", d.Size)))
	return err
}

// GetCurrentSize returns the current disk size in MiB.
func (d *QcowDisk) GetCurrentSize() (int, error) {
	if _, err := os.Stat(d.Path); err != nil {
		return 0, err
	}
	out, err := run(d.QcowTool("info", "--filter", ".size"))
	if err != nil {
		return 0, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, err
	}
	return int(size / mib), nil
}

func (d *QcowDisk) sizeString() string {
	s, err := d.GetCurrentSize()
	if err != nil {
		return fmt.Sprintf("cannot get size: %v", err)
	}
	return fmt.Sprintf("%vMiB", s)
}

// Resize the virtual size of the disk
func (d *QcowDisk) resize() error {
	log.Infof("hyperkit: Resize %q from %v to %dMiB", d, d.sizeString(), d.GetSize())
	_, err := run(d.QcowTool("resize", "--size", fmt.Sprintf("%dMiB", d.Size)))
	return err
}

// compact the disk to shrink the physical size.
func (d *QcowDisk) compact() error {
	log.Infof("hyperkit: Compact: %q... (%v)", d, d.sizeString())
	cmd := d.QcowTool("compact")
	if _, err := run(cmd); err != nil {
		if err.(*exec.ExitError) != nil {
			return errors.New("Failed to compact qcow2")
		}
		return err
	}
	log.Infof("hyperkit: Compact: %q: done (%v)", d, d.sizeString())
	return nil
}

// check the disk is well-formed.
func (d *QcowDisk) check() error {
	cmd := d.QcowTool("check")
	if _, err := run(cmd); err != nil {
		if err.(*exec.ExitError) != nil {
			return errors.New("qcow2 failed integrity check: it may be corrupt")
		}
		return err
	}
	return nil
}

// Stop cleans up this disk when we are quitting.
func (d *QcowDisk) Stop() error {
	if !d.Trim && d.CompactAfter == 0 {
		log.Infof("hyperkit: TRIM is enabled but auto-compaction disabled: compacting %q now", d)
		if err := d.compact(); err != nil {
			return fmt.Errorf("Failed to compact %q: %v", d, err)
		}
		if err := d.check(); err != nil {
			return fmt.Errorf("Post-compact disk integrity check of %q failed: %v", d, err)
		}
		log.Infof("hyperkit: Post-compact disk integrity check of %q successful", d)
	}
	return nil
}

// AsArgument returns the command-line option to pass after `-s <slot>:0,` to hyperkit for this disk.
func (d *QcowDisk) AsArgument() string {
	res := fmt.Sprintf("%s,file://%s?sync=%s&buffered=1", diskDriver(d.Trim), d.Path, d.OnFlush)
	{
		format := d.Format
		if format == "" {
			format = "qcow"
		}
		res += fmt.Sprintf(",format=%v", format)
	}
	if d.Stats != "" {
		res += ",qcow-stats-config=" + d.Stats
	}
	res += fmt.Sprintf(",qcow-config=discard=%t;compact_after_unmaps=%d;keep_erased=%d;runtime_asserts=%t",
		d.Trim, d.CompactAfter, d.KeepErased, d.RuntimeAsserts)
	return res
}
