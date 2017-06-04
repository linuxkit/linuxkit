package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

// Moby is the type of a Moby config file
type Moby struct {
	Kernel struct {
		Image   string
		Cmdline string
	}
	Init     []string
	Onboot   []MobyImage
	Services []MobyImage
	Trust    TrustConfig
	Files    []struct {
		Path      string
		Directory bool
		Symlink   string
		Contents  string
		Source    string
	}
}

// TrustConfig is the type of a content trust config
type TrustConfig struct {
	Image []string
	Org   []string
}

// MobyImage is the type of an image config
type MobyImage struct {
	Name              string             `yaml:"name" json:"name"`
	Image             string             `yaml:"image" json:"image"`
	Capabilities      *[]string          `yaml:"capabilities" json:"capabilities,omitempty"`
	Mounts            *[]specs.Mount     `yaml:"mounts" json:"mounts,omitempty"`
	Binds             *[]string          `yaml:"binds" json:"binds,omitempty"`
	Tmpfs             *[]string          `yaml:"tmpfs" json:"tmpfs,omitempty"`
	Command           *[]string          `yaml:"command" json:"command,omitempty"`
	Env               *[]string          `yaml:"env" json:"env,omitempty"`
	Cwd               string             `yaml:"cwd" json:"cwd"`
	Net               string             `yaml:"net" json:"net"`
	Pid               string             `yaml:"pid" json:"pid"`
	Ipc               string             `yaml:"ipc" json:"ipc"`
	Uts               string             `yaml:"uts" json:"uts"`
	Hostname          string             `yaml:"hostname" json:"hostname"`
	Readonly          *bool              `yaml:"readonly" json:"readonly,omitempty"`
	MaskedPaths       *[]string          `yaml:"maskedPaths" json:"maskedPaths,omitempty"`
	ReadonlyPaths     *[]string          `yaml:"readonlyPaths" json:"readonlyPaths,omitempty"`
	UID               *uint32            `yaml:"uid" json:"uid,omitempty"`
	GID               *uint32            `yaml:"gid" json:"gid,omitempty"`
	AdditionalGids    *[]uint32          `yaml:"additionalGids" json:"additionalGids,omitempty"`
	NoNewPrivileges   *bool              `yaml:"noNewPrivileges" json:"noNewPrivileges,omitempty"`
	OOMScoreAdj       *int               `yaml:"oomScoreAdj" json:"oomScoreAdj,omitempty"`
	DisableOOMKiller  *bool              `yaml:"disableOOMKiller" json:"disableOOMKiller,omitempty"`
	RootfsPropagation *string            `yaml:"rootfsPropagation" json:"rootfsPropagation,omitempty"`
	CgroupsPath       *string            `yaml:"cgroupsPath" json:"cgroupsPath,omitempty"`
	Sysctl            *map[string]string `yaml:"sysctl" json:"sysctl,omitempty"`
}

// github.com/go-yaml/yaml treats map keys as interface{} while encoding/json
// requires them to be strings, integers or to implement encoding.TextMarshaler.
// Fix this up by recursively mapping all map[interface{}]interface{} types into
// map[string]interface{}.
// see http://stackoverflow.com/questions/40737122/convert-yaml-to-json-without-struct-golang#answer-40737676
func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}

// NewConfig parses a config file
func NewConfig(config []byte) (Moby, error) {
	m := Moby{}

	// Parse raw yaml
	var rawYaml interface{}
	err := yaml.Unmarshal(config, &rawYaml)
	if err != nil {
		return m, err
	}

	// Convert to raw JSON
	rawJSON := convert(rawYaml)

	// Validate raw yaml with JSON schema
	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewGoLoader(rawJSON)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return m, err
	}
	if !result.Valid() {
		fmt.Printf("The configuration file is invalid:\n")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
		return m, fmt.Errorf("invalid configuration file")
	}

	// Parse yaml
	err = yaml.Unmarshal(config, &m)
	if err != nil {
		return m, err
	}

	return m, nil
}

// AppendConfig appends two configs.
func AppendConfig(m0, m1 Moby) Moby {
	moby := m0
	if m1.Kernel.Image != "" {
		moby.Kernel.Image = m1.Kernel.Image
	}
	if m1.Kernel.Cmdline != "" {
		moby.Kernel.Cmdline = m1.Kernel.Cmdline
	}
	moby.Init = append(moby.Init, m1.Init...)
	moby.Onboot = append(moby.Onboot, m1.Onboot...)
	moby.Services = append(moby.Services, m1.Services...)
	moby.Files = append(moby.Files, m1.Files...)
	moby.Trust.Image = append(moby.Trust.Image, m1.Trust.Image...)
	moby.Trust.Org = append(moby.Trust.Org, m1.Trust.Org...)

	return moby
}

// NewImage validates an parses yaml or json for a MobyImage
func NewImage(config []byte) (MobyImage, error) {
	log.Debugf("Reading label config: %s", string(config))

	mi := MobyImage{}

	// Parse raw yaml
	var rawYaml interface{}
	err := yaml.Unmarshal(config, &rawYaml)
	if err != nil {
		return mi, err
	}

	// Convert to raw JSON
	rawJSON := convert(rawYaml)

	// check it is an object not an array
	jsonObject, ok := rawJSON.(map[string]interface{})
	if !ok {
		return mi, fmt.Errorf("JSON is an array not an object: %s", string(config))
	}

	// add a dummy name and image to pass validation
	var dummyName interface{}
	var dummyImage interface{}
	dummyName = "dummyname"
	dummyImage = "dummyimage"
	jsonObject["name"] = dummyName
	jsonObject["image"] = dummyImage

	// Validate it as {"services": [config]}
	var services [1]interface{}
	services[0] = rawJSON
	serviceJSON := map[string]interface{}{"services": services}

	// Validate serviceJSON with JSON schema
	schemaLoader := gojsonschema.NewStringLoader(schema)
	documentLoader := gojsonschema.NewGoLoader(serviceJSON)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return mi, err
	}
	if !result.Valid() {
		fmt.Printf("The org.mobyproject.config label is invalid:\n")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
		return mi, fmt.Errorf("invalid configuration label")
	}

	// Parse yaml
	err = yaml.Unmarshal(config, &mi)
	if err != nil {
		return mi, err
	}

	if mi.Name != "" {
		return mi, fmt.Errorf("name cannot be set in metadata label")
	}
	if mi.Image != "" {
		return mi, fmt.Errorf("image cannot be set in metadata label")
	}

	return mi, nil
}

// ConfigToOCI converts a config specification to an OCI config file
func ConfigToOCI(image MobyImage) ([]byte, error) {

	// TODO pass through same docker client to all functions
	cli, err := dockerClient()
	if err != nil {
		return []byte{}, err
	}

	inspect, err := dockerInspectImage(cli, image.Image)
	if err != nil {
		return []byte{}, err
	}

	oci, err := ConfigInspectToOCI(image, inspect)
	if err != nil {
		return []byte{}, err
	}

	return json.MarshalIndent(oci, "", "    ")
}

func defaultMountpoint(tp string) string {
	switch tp {
	case "proc":
		return "/proc"
	case "devpts":
		return "/dev/pts"
	case "sysfs":
		return "/sys"
	case "cgroup":
		return "/sys/fs/cgroup"
	case "mqueue":
		return "/dev/mqueue"
	default:
		return ""
	}
}

// Sort mounts by number of path components so /dev/pts is listed after /dev
type mlist []specs.Mount

func (m mlist) Len() int {
	return len(m)
}
func (m mlist) Less(i, j int) bool {
	return m.parts(i) < m.parts(j)
}
func (m mlist) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}
func (m mlist) parts(i int) int {
	return strings.Count(filepath.Clean(m[i].Destination), string(os.PathSeparator))
}

// assignBool does ordered overrides from JSON bool pointers
func assignBool(v1, v2 *bool) bool {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return false
}

// assignBoolPtr does ordered overrides from JSON bool pointers
func assignBoolPtr(v1, v2 *bool) *bool {
	if v2 != nil {
		return v2
	}
	if v1 != nil {
		return v1
	}
	return nil
}

// assignIntPtr does ordered overrides from JSON int pointers
func assignIntPtr(v1, v2 *int) *int {
	if v2 != nil {
		return v2
	}
	if v1 != nil {
		return v1
	}
	return nil
}

// assignUint32 does ordered overrides from JSON uint32 pointers
func assignUint32(v1, v2 *uint32) uint32 {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return 0
}

// assignUint32Array does ordered overrides from JSON uint32 array pointers
func assignUint32Array(v1, v2 *[]uint32) []uint32 {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return []uint32{}
}

// assignStrings does ordered overrides from JSON string array pointers
func assignStrings(v1, v2 *[]string) []string {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return []string{}
}

// assignStrings3 does ordered overrides from JSON string array pointers
func assignStrings3(v1 []string, v2, v3 *[]string) []string {
	if v3 != nil {
		return *v3
	}
	if v2 != nil {
		return *v2
	}
	return v1
}

// assignMaps does ordered overrides from JSON string map pointers
func assignMaps(v1, v2 *map[string]string) map[string]string {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return map[string]string{}
}

// assignBinds does ordered overrides from JSON Bind array pointers
func assignBinds(v1, v2 *[]specs.Mount) []specs.Mount {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return []specs.Mount{}
}

// assignString does ordered overrides from JSON string pointers
func assignString(v1, v2 *string) string {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return ""
}

// assignStringEmpty does ordered overrides if strings are empty, for
// values where there is always an explicit override eg "none"
func assignStringEmpty(v1, v2 string) string {
	if v2 != "" {
		return v2
	}
	return v1
}

// assignStringEmpty3 does ordered overrides if strings are empty, for
// values where there is always an explicit override eg "none"
func assignStringEmpty3(v1, v2, v3 string) string {
	if v3 != "" {
		return v3
	}
	if v2 != "" {
		return v2
	}
	return v1
}

// assign StringEmpty4 does ordered overrides if strings are empty, for
// values where there is always an explicit override eg "none"
func assignStringEmpty4(v1, v2, v3, v4 string) string {
	if v4 != "" {
		return v4
	}
	if v3 != "" {
		return v3
	}
	if v2 != "" {
		return v2
	}
	return v1
}

// ConfigInspectToOCI converts a config and the output of image inspect to an OCI config
func ConfigInspectToOCI(yaml MobyImage, inspect types.ImageInspect) (specs.Spec, error) {
	oci := specs.Spec{}

	var inspectConfig container.Config
	if inspect.Config != nil {
		inspectConfig = *inspect.Config
	}

	// look for org.mobyproject.config label
	var label MobyImage
	labelString := inspectConfig.Labels["org.mobyproject.config"]
	if labelString != "" {
		var err error
		label, err = NewImage([]byte(labelString))
		if err != nil {
			return oci, err
		}
	}

	// command, env and cwd can be taken from image, as they are commonly specified in Dockerfile

	// TODO we could handle entrypoint and cmd independently more like Docker
	inspectCommand := append(inspectConfig.Entrypoint, inspect.Config.Cmd...)
	args := assignStrings3(inspectCommand, label.Command, yaml.Command)

	env := assignStrings3(inspectConfig.Env, label.Env, yaml.Env)

	// empty Cwd not allowed in OCI, must be / in that case
	cwd := assignStringEmpty4("/", inspectConfig.WorkingDir, label.Cwd, yaml.Cwd)

	// the other options will never be in the image config, but may be in label or yaml

	readonly := assignBool(label.Readonly, yaml.Readonly)

	// default options match what Docker does
	procOptions := []string{"nosuid", "nodev", "noexec", "relatime"}
	devOptions := []string{"nosuid", "strictatime", "mode=755", "size=65536k"}
	if readonly {
		devOptions = append(devOptions, "ro")
	}
	ptsOptions := []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"}
	sysOptions := []string{"nosuid", "noexec", "nodev"}
	if readonly {
		sysOptions = append(sysOptions, "ro")
	}
	cgroupOptions := []string{"nosuid", "noexec", "nodev", "relatime", "ro"}
	// note omits "standard" /dev/shm and /dev/mqueue
	mounts := map[string]specs.Mount{
		"/proc":          {Destination: "/proc", Type: "proc", Source: "proc", Options: procOptions},
		"/dev":           {Destination: "/dev", Type: "tmpfs", Source: "tmpfs", Options: devOptions},
		"/dev/pts":       {Destination: "/dev/pts", Type: "devpts", Source: "devpts", Options: ptsOptions},
		"/sys":           {Destination: "/sys", Type: "sysfs", Source: "sysfs", Options: sysOptions},
		"/sys/fs/cgroup": {Destination: "/sys/fs/cgroup", Type: "cgroup", Source: "cgroup", Options: cgroupOptions},
	}
	for _, t := range assignStrings(label.Tmpfs, yaml.Tmpfs) {
		parts := strings.Split(t, ":")
		if len(parts) > 2 {
			return oci, fmt.Errorf("Cannot parse tmpfs, too many ':': %s", t)
		}
		dest := parts[0]
		opts := []string{}
		if len(parts) == 2 {
			opts = strings.Split(parts[1], ",")
		}
		mounts[dest] = specs.Mount{Destination: dest, Type: "tmpfs", Source: "tmpfs", Options: opts}
	}
	for _, b := range assignStrings(label.Binds, yaml.Binds) {
		parts := strings.Split(b, ":")
		if len(parts) < 2 {
			return oci, fmt.Errorf("Cannot parse bind, missing ':': %s", b)
		}
		if len(parts) > 3 {
			return oci, fmt.Errorf("Cannot parse bind, too many ':': %s", b)
		}
		src := parts[0]
		dest := parts[1]
		opts := []string{"rw", "rbind", "rprivate"}
		if len(parts) == 3 {
			opts = append(strings.Split(parts[2], ","), "rbind")
		}
		mounts[dest] = specs.Mount{Destination: dest, Type: "bind", Source: src, Options: opts}
	}
	for _, m := range assignBinds(label.Mounts, yaml.Mounts) {
		tp := m.Type
		src := m.Source
		dest := m.Destination
		opts := m.Options
		if tp == "" {
			switch src {
			case "mqueue", "devpts", "proc", "sysfs", "cgroup":
				tp = src
			}
		}
		if tp == "" && dest == "/dev" {
			tp = "tmpfs"
		}
		if tp == "" {
			return oci, fmt.Errorf("Mount for destination %s is missing type", dest)
		}
		if src == "" {
			// usually sane, eg proc, tmpfs etc
			src = tp
		}
		if dest == "" {
			dest = defaultMountpoint(tp)
		}
		if dest == "" {
			return oci, fmt.Errorf("Mount type %s is missing destination", tp)
		}
		mounts[dest] = specs.Mount{Destination: dest, Type: tp, Source: src, Options: opts}
	}
	mountList := mlist{}
	for _, m := range mounts {
		mountList = append(mountList, m)
	}
	sort.Sort(mountList)

	namespaces := []specs.LinuxNamespace{}
	// to attach to an existing namespace, easiest to bind mount with nsfs in a system container

	// net, ipc and uts namespaces: default to not creating a new namespace (usually host namespace)
	netNS := assignStringEmpty3("root", label.Net, yaml.Net)
	if netNS != "host" && netNS != "root" {
		if netNS == "none" || netNS == "new" {
			netNS = ""
		}
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.NetworkNamespace, Path: netNS})
	}
	ipcNS := assignStringEmpty3("root", label.Ipc, yaml.Ipc)
	if ipcNS != "host" && ipcNS != "root" {
		if ipcNS == "new" {
			ipcNS = ""
		}
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.IPCNamespace, Path: ipcNS})
	}
	utsNS := assignStringEmpty3("root", label.Uts, yaml.Uts)
	if utsNS != "host" && utsNS != "root" {
		if utsNS == "new" {
			utsNS = ""
		}
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.UTSNamespace, Path: utsNS})
	}

	// default to creating a new pid namespace
	pidNS := assignStringEmpty(label.Pid, yaml.Pid)
	if pidNS != "host" && pidNS != "root" {
		if pidNS == "new" {
			pidNS = ""
		}
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.PIDNamespace, Path: pidNS})
	}

	// Always create a new mount namespace
	namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.MountNamespace})

	// TODO user, cgroup namespaces

	caps := assignStrings(label.Capabilities, yaml.Capabilities)
	if len(caps) == 1 {
		switch cap := strings.ToLower(caps[0]); cap {
		case "none":
			caps = []string{}
		case "all":
			caps = []string{
				"CAP_AUDIT_CONTROL",
				"CAP_AUDIT_READ",
				"CAP_AUDIT_WRITE",
				"CAP_BLOCK_SUSPEND",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_IPC_LOCK",
				"CAP_IPC_OWNER",
				"CAP_KILL",
				"CAP_LEASE",
				"CAP_LINUX_IMMUTABLE",
				"CAP_MAC_ADMIN",
				"CAP_MAC_OVERRIDE",
				"CAP_MKNOD",
				"CAP_NET_ADMIN",
				"CAP_NET_BIND_SERVICE",
				"CAP_NET_BROADCAST",
				"CAP_NET_RAW",
				"CAP_SETFCAP",
				"CAP_SETGID",
				"CAP_SETPCAP",
				"CAP_SETUID",
				"CAP_SYSLOG",
				"CAP_SYS_ADMIN",
				"CAP_SYS_BOOT",
				"CAP_SYS_CHROOT",
				"CAP_SYS_MODULE",
				"CAP_SYS_NICE",
				"CAP_SYS_PACCT",
				"CAP_SYS_PTRACE",
				"CAP_SYS_RAWIO",
				"CAP_SYS_RESOURCE",
				"CAP_SYS_TIME",
				"CAP_SYS_TTY_CONFIG",
				"CAP_WAKE_ALARM",
			}
		}
	}

	oci.Version = specs.Version

	oci.Platform = specs.Platform{
		OS:   inspect.Os,
		Arch: inspect.Architecture,
	}

	oci.Process = specs.Process{
		Terminal: false,
		//ConsoleSize
		User: specs.User{
			UID:            assignUint32(label.UID, yaml.UID),
			GID:            assignUint32(label.GID, yaml.GID),
			AdditionalGids: assignUint32Array(label.AdditionalGids, yaml.AdditionalGids),
			// Username (Windows)
		},
		Args: args,
		Env:  env,
		Cwd:  cwd,
		Capabilities: &specs.LinuxCapabilities{
			Bounding:    caps,
			Effective:   caps,
			Inheritable: caps,
			Permitted:   caps,
			Ambient:     []string{},
		},
		Rlimits:         []specs.LinuxRlimit{},
		NoNewPrivileges: assignBool(label.NoNewPrivileges, yaml.NoNewPrivileges),
		// ApparmorProfile
		// TODO FIXME this has moved in runc spec and needs a revendor and update
		//OOMScoreAdj: assignIntPtr(label.OOMScoreAdj, yaml.OOMScoreAdj),
		// SelinuxLabel
	}

	oci.Root = specs.Root{
		Path:     "rootfs",
		Readonly: readonly,
	}

	oci.Hostname = assignStringEmpty(label.Hostname, yaml.Hostname)
	oci.Mounts = mountList

	oci.Linux = &specs.Linux{
		// UIDMappings
		// GIDMappings
		Sysctl: assignMaps(label.Sysctl, yaml.Sysctl),
		Resources: &specs.LinuxResources{
			// Devices
			DisableOOMKiller: assignBoolPtr(label.DisableOOMKiller, yaml.DisableOOMKiller),
			// Memory
			// CPU
			// Pids
			// BlockIO
			// HugepageLimits
			// Network
		},
		CgroupsPath: assignString(label.CgroupsPath, yaml.CgroupsPath),
		Namespaces:  namespaces,
		// Devices
		// Seccomp
		RootfsPropagation: assignString(label.RootfsPropagation, yaml.RootfsPropagation),
		MaskedPaths:       assignStrings(label.MaskedPaths, yaml.MaskedPaths),
		ReadonlyPaths:     assignStrings(label.ReadonlyPaths, yaml.ReadonlyPaths),
		// MountLabel
		// IntelRdt
	}

	return oci, nil
}

func filesystem(m Moby) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	if len(m.Files) != 0 {
		log.Infof("Add files:")
	}
	for _, f := range m.Files {
		log.Infof("  %s", f.Path)
		if f.Path == "" {
			return buf, errors.New("Did not specify path for file")
		}
		if !f.Directory && f.Contents == "" && f.Symlink == "" {
			if f.Source == "" {
				return buf, errors.New("Contents of file not specified")
			}

			contents, err := ioutil.ReadFile(f.Source)
			if err != nil {
				return buf, err
			}

			f.Contents = string(contents)
		}
		// we need all the leading directories
		parts := strings.Split(path.Dir(f.Path), "/")
		root := ""
		for _, p := range parts {
			if p == "." || p == "/" {
				continue
			}
			if root == "" {
				root = p
			} else {
				root = root + "/" + p
			}
			hdr := &tar.Header{
				Name:     root,
				Typeflag: tar.TypeDir,
				Mode:     0700,
			}
			err := tw.WriteHeader(hdr)
			if err != nil {
				return buf, err
			}
		}

		if f.Directory {
			if f.Contents != "" {
				return buf, errors.New("Directory with contents not allowed")
			}
			hdr := &tar.Header{
				Name:     f.Path,
				Typeflag: tar.TypeDir,
				Mode:     0700,
			}
			err := tw.WriteHeader(hdr)
			if err != nil {
				return buf, err
			}
		} else if f.Symlink != "" {
			hdr := &tar.Header{
				Name:     f.Path,
				Typeflag: tar.TypeSymlink,
				Mode:     0600,
				Linkname: f.Symlink,
			}
			err := tw.WriteHeader(hdr)
			if err != nil {
				return buf, err
			}
		} else {
			hdr := &tar.Header{
				Name: f.Path,
				Mode: 0600,
				Size: int64(len(f.Contents)),
			}
			err := tw.WriteHeader(hdr)
			if err != nil {
				return buf, err
			}
			_, err = tw.Write([]byte(f.Contents))
			if err != nil {
				return buf, err
			}
		}
	}
	return buf, nil
}
