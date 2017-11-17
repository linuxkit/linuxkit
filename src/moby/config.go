package moby

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/containerd/containerd/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

// Moby is the type of a Moby config file
type Moby struct {
	Kernel     KernelConfig `kernel:"cmdline" json:"kernel,omitempty"`
	Init       []string     `init:"cmdline" json:"init"`
	Onboot     []*Image     `yaml:"onboot" json:"onboot"`
	Onshutdown []*Image     `yaml:"onshutdown" json:"onshutdown"`
	Services   []*Image     `yaml:"services" json:"services"`
	Trust      TrustConfig  `yaml:"trust" json:"trust,omitempty"`
	Files      []File       `yaml:"files" json:"files"`

	initRefs []*reference.Spec
}

// KernelConfig is the type of the config for a kernel
type KernelConfig struct {
	Image   string  `yaml:"image" json:"image"`
	Cmdline string  `yaml:"cmdline" json:"cmdline,omitempty"`
	Binary  string  `yaml:"binary" json:"binary,omitempty"`
	Tar     *string `yaml:"tar" json:"tar,omitempty"`

	ref *reference.Spec
}

// TrustConfig is the type of a content trust config
type TrustConfig struct {
	Image []string `yaml:"image" json:"image,omitempty"`
	Org   []string `yaml:"org" json:"org,omitempty"`
}

// File is the type of a file specification
type File struct {
	Path      string      `yaml:"path" json:"path"`
	Directory bool        `yaml:"directory" json:"directory"`
	Symlink   string      `yaml:"symlink" json:"symlink,omitempty"`
	Contents  *string     `yaml:"contents" json:"contents,omitempty"`
	Source    string      `yaml:"source" json:"source,omitempty"`
	Metadata  string      `yaml:"metadata" json:"metadata,omitempty"`
	Optional  bool        `yaml:"optional" json:"optional"`
	Mode      string      `yaml:"mode" json:"mode,omitempty"`
	UID       interface{} `yaml:"uid" json:"uid,omitempty"`
	GID       interface{} `yaml:"gid" json:"gid,omitempty"`
}

// Image is the type of an image config
type Image struct {
	Name              string                  `yaml:"name" json:"name"`
	Image             string                  `yaml:"image" json:"image"`
	Capabilities      *[]string               `yaml:"capabilities" json:"capabilities,omitempty"`
	Ambient           *[]string               `yaml:"ambient" json:"ambient,omitempty"`
	Mounts            *[]specs.Mount          `yaml:"mounts" json:"mounts,omitempty"`
	Binds             *[]string               `yaml:"binds" json:"binds,omitempty"`
	Tmpfs             *[]string               `yaml:"tmpfs" json:"tmpfs,omitempty"`
	Command           *[]string               `yaml:"command" json:"command,omitempty"`
	Env               *[]string               `yaml:"env" json:"env,omitempty"`
	Cwd               string                  `yaml:"cwd" json:"cwd,omitempty"`
	Net               string                  `yaml:"net" json:"net,omitempty"`
	Pid               string                  `yaml:"pid" json:"pid,omitempty"`
	Ipc               string                  `yaml:"ipc" json:"ipc,omitempty"`
	Uts               string                  `yaml:"uts" json:"uts,omitempty"`
	Userns            string                  `yaml:"userns" json:"userns,omitempty"`
	Hostname          string                  `yaml:"hostname" json:"hostname,omitempty"`
	Readonly          *bool                   `yaml:"readonly" json:"readonly,omitempty"`
	MaskedPaths       *[]string               `yaml:"maskedPaths" json:"maskedPaths,omitempty"`
	ReadonlyPaths     *[]string               `yaml:"readonlyPaths" json:"readonlyPaths,omitempty"`
	UID               *interface{}            `yaml:"uid" json:"uid,omitempty"`
	GID               *interface{}            `yaml:"gid" json:"gid,omitempty"`
	AdditionalGids    *[]interface{}          `yaml:"additionalGids" json:"additionalGids,omitempty"`
	NoNewPrivileges   *bool                   `yaml:"noNewPrivileges" json:"noNewPrivileges,omitempty"`
	OOMScoreAdj       *int                    `yaml:"oomScoreAdj" json:"oomScoreAdj,omitempty"`
	RootfsPropagation *string                 `yaml:"rootfsPropagation" json:"rootfsPropagation,omitempty"`
	CgroupsPath       *string                 `yaml:"cgroupsPath" json:"cgroupsPath,omitempty"`
	Resources         *specs.LinuxResources   `yaml:"resources" json:"resources,omitempty"`
	Sysctl            *map[string]string      `yaml:"sysctl" json:"sysctl,omitempty"`
	Rlimits           *[]string               `yaml:"rlimits" json:"rlimits,omitempty"`
	UIDMappings       *[]specs.LinuxIDMapping `yaml:"uidMappings" json:"uidMappings,omitempty"`
	GIDMappings       *[]specs.LinuxIDMapping `yaml:"gidMappings" json:"gidMappings,omitempty"`
	Runtime           *Runtime                `yaml:"runtime" json:"runtime,omitempty"`

	ref *reference.Spec
}

// Runtime is the type of config processed at runtime, not used to build the OCI spec
type Runtime struct {
	Cgroups    *[]string      `yaml:"cgroups" json:"cgroups,omitempty"`
	Mounts     *[]specs.Mount `yaml:"mounts" json:"mounts,omitempty"`
	Mkdir      *[]string      `yaml:"mkdir" json:"mkdir,omitempty"`
	Interfaces *[]Interface   `yaml:"interfaces" json:"interfaces,omitempty"`
	BindNS     Namespaces     `yaml:"bindNS" json:"bindNS,omitempty"`
}

// Namespaces is the type for configuring paths to bind namespaces
type Namespaces struct {
	Cgroup *string `yaml:"cgroup" json:"cgroup,omitempty"`
	Ipc    *string `yaml:"ipc" json:"ipc,omitempty"`
	Mnt    *string `yaml:"mnt" json:"mnt,omitempty"`
	Net    *string `yaml:"net" json:"net,omitempty"`
	Pid    *string `yaml:"pid" json:"pid,omitempty"`
	User   *string `yaml:"user" json:"user,omitempty"`
	Uts    *string `yaml:"uts" json:"uts,omitempty"`
}

// Interface is the runtime config for network interfaces
type Interface struct {
	Name         string `yaml:"name" json:"name,omitempty"`
	Add          string `yaml:"add" json:"add,omitempty"`
	Peer         string `yaml:"peer" json:"peer,omitempty"`
	CreateInRoot bool   `yaml:"createInRoot" json:"createInRoot"`
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

func uniqueServices(m Moby) error {
	// service names must be unique, as they run as simultaneous containers
	names := map[string]bool{}
	for _, s := range m.Services {
		if names[s.Name] {
			return fmt.Errorf("duplicate service name: %s", s.Name)
		}
		names[s.Name] = true
	}
	return nil
}

func extractReferences(m *Moby) error {
	if m.Kernel.Image != "" {
		r, err := reference.Parse(m.Kernel.Image)
		if err != nil {
			return fmt.Errorf("extract kernel image reference: %v", err)
		}
		m.Kernel.ref = &r
	}
	for _, ii := range m.Init {
		r, err := reference.Parse(ii)
		if err != nil {
			return fmt.Errorf("extract on boot image reference: %v", err)
		}
		m.initRefs = append(m.initRefs, &r)
	}
	for _, image := range m.Onboot {
		r, err := reference.Parse(image.Image)
		if err != nil {
			return fmt.Errorf("extract on boot image reference: %v", err)
		}
		image.ref = &r
	}
	for _, image := range m.Onshutdown {
		r, err := reference.Parse(image.Image)
		if err != nil {
			return fmt.Errorf("extract on shutdown image reference: %v", err)
		}
		image.ref = &r
	}
	for _, image := range m.Services {
		r, err := reference.Parse(image.Image)
		if err != nil {
			return fmt.Errorf("extract service image reference: %v", err)
		}
		image.ref = &r
	}
	return nil
}

func updateImages(m *Moby) {
	if m.Kernel.ref != nil {
		m.Kernel.Image = m.Kernel.ref.String()
	}
	for i, ii := range m.initRefs {
		m.Init[i] = ii.String()
	}
	for _, image := range m.Onboot {
		if image.ref != nil {
			image.Image = image.ref.String()
		}
	}
	for _, image := range m.Onshutdown {
		if image.ref != nil {
			image.Image = image.ref.String()
		}
	}
	for _, image := range m.Services {
		if image.ref != nil {
			image.Image = image.ref.String()
		}
	}
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

	if err := uniqueServices(m); err != nil {
		return m, err
	}

	if err := extractReferences(&m); err != nil {
		return m, err
	}

	return m, nil
}

// AppendConfig appends two configs.
func AppendConfig(m0, m1 Moby) (Moby, error) {
	moby := m0
	if m1.Kernel.Image != "" {
		moby.Kernel.Image = m1.Kernel.Image
	}
	if m1.Kernel.Cmdline != "" {
		moby.Kernel.Cmdline = m1.Kernel.Cmdline
	}
	if m1.Kernel.Binary != "" {
		moby.Kernel.Binary = m1.Kernel.Binary
	}
	if m1.Kernel.Tar != nil {
		moby.Kernel.Tar = m1.Kernel.Tar
	}
	if m1.Kernel.ref != nil {
		moby.Kernel.ref = m1.Kernel.ref
	}
	moby.Init = append(moby.Init, m1.Init...)
	moby.Onboot = append(moby.Onboot, m1.Onboot...)
	moby.Onshutdown = append(moby.Onshutdown, m1.Onshutdown...)
	moby.Services = append(moby.Services, m1.Services...)
	moby.Files = append(moby.Files, m1.Files...)
	moby.Trust.Image = append(moby.Trust.Image, m1.Trust.Image...)
	moby.Trust.Org = append(moby.Trust.Org, m1.Trust.Org...)
	moby.initRefs = append(moby.initRefs, m1.initRefs...)

	return moby, uniqueServices(moby)
}

// NewImage validates an parses yaml or json for a Image
func NewImage(config []byte) (Image, error) {
	log.Debugf("Reading label config: %s", string(config))

	mi := Image{}

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

// ConfigToOCI converts a config specification to an OCI config file and a runtime config
func ConfigToOCI(image *Image, trust bool, idMap map[string]uint32) (specs.Spec, Runtime, error) {

	// TODO pass through same docker client to all functions
	cli, err := dockerClient()
	if err != nil {
		return specs.Spec{}, Runtime{}, err
	}
	inspect, err := dockerInspectImage(cli, image.ref, trust)
	if err != nil {
		return specs.Spec{}, Runtime{}, err
	}

	oci, runtime, err := ConfigInspectToOCI(image, inspect, idMap)
	if err != nil {
		return specs.Spec{}, Runtime{}, err
	}

	return oci, runtime, nil
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

// assignInterface does ordered overrides from Go interfaces
// we return 0 as we are using this for uid and this is the default
func assignInterface(v1, v2 *interface{}) interface{} {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return 0
}

// assignInterfaceArray does ordered overrides from arrays of Go interfaces
func assignInterfaceArray(v1, v2 *[]interface{}) []interface{} {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return []interface{}{}
}

// assignRuntimeInterfaceArray does ordered overrides from arrays of Interface structs
func assignRuntimeInterfaceArray(v1, v2 *[]Interface) []Interface {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return []Interface{}
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

// assignString does ordered overrides from JSON string pointers
func assignStringPtr(v1, v2 *string) *string {
	if v2 != nil {
		return v2
	}
	if v1 != nil {
		return v1
	}
	s := ""
	return &s
}

// assignMappings does ordered overrides from UID, GID maps
func assignMappings(v1, v2 *[]specs.LinuxIDMapping) []specs.LinuxIDMapping {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return []specs.LinuxIDMapping{}
}

// assignResources does ordered overrides from Resources
func assignResources(v1, v2 *specs.LinuxResources) specs.LinuxResources {
	if v2 != nil {
		return *v2
	}
	if v1 != nil {
		return *v1
	}
	return specs.LinuxResources{}
}

// assignRuntime does ordered overrides from Runtime
func assignRuntime(v1, v2 *Runtime) Runtime {
	if v1 == nil {
		v1 = &Runtime{}
	}
	if v2 == nil {
		v2 = &Runtime{}
	}
	runtimeCgroups := assignStrings(v1.Cgroups, v2.Cgroups)
	runtimeMounts := assignBinds(v1.Mounts, v2.Mounts)
	runtimeMkdir := assignStrings(v1.Mkdir, v2.Mkdir)
	runtimeInterfaces := assignRuntimeInterfaceArray(v1.Interfaces, v2.Interfaces)
	runtime := Runtime{
		Cgroups:    &runtimeCgroups,
		Mounts:     &runtimeMounts,
		Mkdir:      &runtimeMkdir,
		Interfaces: &runtimeInterfaces,
		BindNS: Namespaces{
			Cgroup: assignStringPtr(v1.BindNS.Cgroup, v2.BindNS.Cgroup),
			Ipc:    assignStringPtr(v1.BindNS.Ipc, v2.BindNS.Ipc),
			Mnt:    assignStringPtr(v1.BindNS.Mnt, v2.BindNS.Mnt),
			Net:    assignStringPtr(v1.BindNS.Net, v2.BindNS.Net),
			Pid:    assignStringPtr(v1.BindNS.Pid, v2.BindNS.Pid),
			User:   assignStringPtr(v1.BindNS.User, v2.BindNS.User),
			Uts:    assignStringPtr(v1.BindNS.Uts, v2.BindNS.Uts),
		},
	}
	return runtime
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

var allCaps = []string{
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

func idNumeric(v interface{}, idMap map[string]uint32) (uint32, error) {
	switch id := v.(type) {
	case nil:
		return uint32(0), nil
	case int:
		return uint32(id), nil
	case string:
		if id == "" || id == "root" {
			return uint32(0), nil
		}
		for k, v := range idMap {
			if id == k {
				return v, nil
			}
		}
		return 0, fmt.Errorf("Cannot find id: %s", id)
	default:
		return 0, fmt.Errorf("Bad type for uid or gid")
	}
}

// ConfigInspectToOCI converts a config and the output of image inspect to an OCI config
func ConfigInspectToOCI(yaml *Image, inspect types.ImageInspect, idMap map[string]uint32) (specs.Spec, Runtime, error) {
	oci := specs.Spec{}
	runtime := Runtime{}

	var inspectConfig container.Config
	if inspect.Config != nil {
		inspectConfig = *inspect.Config
	}

	// look for org.mobyproject.config label
	var label Image
	labelString := inspectConfig.Labels["org.mobyproject.config"]
	if labelString != "" {
		var err error
		label, err = NewImage([]byte(labelString))
		if err != nil {
			return oci, runtime, err
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
			return oci, runtime, fmt.Errorf("Cannot parse tmpfs, too many ':': %s", t)
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
			return oci, runtime, fmt.Errorf("Cannot parse bind, missing ':': %s", b)
		}
		if len(parts) > 3 {
			return oci, runtime, fmt.Errorf("Cannot parse bind, too many ':': %s", b)
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
			return oci, runtime, fmt.Errorf("Mount for destination %s is missing type", dest)
		}
		if src == "" {
			// usually sane, eg proc, tmpfs etc
			src = tp
		}
		if dest == "" {
			dest = defaultMountpoint(tp)
		}
		if dest == "" {
			return oci, runtime, fmt.Errorf("Mount type %s is missing destination", tp)
		}
		mounts[dest] = specs.Mount{Destination: dest, Type: tp, Source: src, Options: opts}
	}
	mountList := mlist{}
	for _, m := range mounts {
		mountList = append(mountList, m)
	}
	sort.Sort(mountList)

	namespaces := []specs.LinuxNamespace{}

	// net, ipc, and uts namespaces: default to not creating a new namespace (usually host namespace)
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

	// do not create a user namespace unless asked, needs additional configuration
	userNS := assignStringEmpty3("root", label.Userns, yaml.Userns)
	if userNS != "host" && userNS != "root" {
		if userNS == "new" {
			userNS = ""
		}
		namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.UserNamespace, Path: userNS})
	}

	// Always create a new mount namespace
	namespaces = append(namespaces, specs.LinuxNamespace{Type: specs.MountNamespace})

	// TODO cgroup namespaces

	// Capabilities
	capCheck := map[string]bool{}
	for _, capability := range allCaps {
		capCheck[capability] = true
	}
	boundingSet := map[string]bool{}
	caps := assignStrings(label.Capabilities, yaml.Capabilities)
	if len(caps) == 1 {
		switch cap := strings.ToLower(caps[0]); cap {
		case "none":
			caps = []string{}
		case "all":
			caps = allCaps[:]
		}
	}
	for _, capability := range caps {
		if !capCheck[capability] {
			return oci, runtime, fmt.Errorf("unknown capability: %s", capability)
		}
		boundingSet[capability] = true
	}
	ambient := assignStrings(label.Ambient, yaml.Ambient)
	if len(ambient) == 1 {
		switch cap := strings.ToLower(ambient[0]); cap {
		case "none":
			ambient = []string{}
		case "all":
			ambient = allCaps[:]
		}
	}
	for _, capability := range ambient {
		if !capCheck[capability] {
			return oci, runtime, fmt.Errorf("unknown capability: %s", capability)
		}
		boundingSet[capability] = true
	}
	bounding := []string{}
	for capability := range boundingSet {
		bounding = append(bounding, capability)
	}

	rlimitsString := assignStrings(label.Rlimits, yaml.Rlimits)
	rlimits := []specs.POSIXRlimit{}
	for _, limitString := range rlimitsString {
		rs := strings.SplitN(limitString, ",", 3)
		var limit string
		var soft, hard uint64
		switch len(rs) {
		case 3:
			origLimit := limit
			limit = strings.ToUpper(strings.TrimSpace(rs[0]))
			if !strings.HasPrefix(limit, "RLIMIT_") {
				limit = "RLIMIT_" + limit
			}
			softString := strings.TrimSpace(rs[1])
			if strings.ToLower(softString) == "unlimited" {
				soft = 18446744073709551615
			} else {
				var err error
				soft, err = strconv.ParseUint(softString, 10, 64)
				if err != nil {
					return oci, runtime, fmt.Errorf("Cannot parse %s as uint64: %v", softString, err)
				}
			}
			hardString := strings.TrimSpace(rs[2])
			if strings.ToLower(hardString) == "unlimited" {
				hard = 18446744073709551615
			} else {
				var err error
				hard, err = strconv.ParseUint(hardString, 10, 64)
				if err != nil {
					return oci, runtime, fmt.Errorf("Cannot parse %s as uint64: %v", hardString, err)
				}
			}
			switch limit {
			case
				"RLIMIT_CPU",
				"RLIMIT_FSIZE",
				"RLIMIT_DATA",
				"RLIMIT_STACK",
				"RLIMIT_CORE",
				"RLIMIT_RSS",
				"RLIMIT_NPROC",
				"RLIMIT_NOFILE",
				"RLIMIT_MEMLOCK",
				"RLIMIT_AS",
				"RLIMIT_LOCKS",
				"RLIMIT_SIGPENDING",
				"RLIMIT_MSGQUEUE",
				"RLIMIT_NICE",
				"RLIMIT_RTPRIO",
				"RLIMIT_RTTIME":
				rlimits = append(rlimits, specs.POSIXRlimit{Type: limit, Soft: soft, Hard: hard})
			default:
				return oci, runtime, fmt.Errorf("Unknown limit: %s", origLimit)
			}
		default:
			return oci, runtime, fmt.Errorf("Cannot parse rlimit: %s", rlimitsString)
		}
	}

	// handle mapping of named uid, gid to numbers
	uidIf := assignInterface(label.UID, yaml.UID)
	gidIf := assignInterface(label.GID, yaml.GID)
	agIf := assignInterfaceArray(label.AdditionalGids, yaml.AdditionalGids)
	uid, err := idNumeric(uidIf, idMap)
	if err != nil {
		return oci, runtime, err
	}
	gid, err := idNumeric(gidIf, idMap)
	if err != nil {
		return oci, runtime, err
	}
	additionalGroups := []uint32{}
	for _, id := range agIf {
		ag, err := idNumeric(id, idMap)
		if err != nil {
			return oci, runtime, err
		}
		additionalGroups = append(additionalGroups, ag)
	}

	oci.Version = specs.Version

	oci.Process = &specs.Process{
		Terminal: false,
		//ConsoleSize
		User: specs.User{
			UID:            uid,
			GID:            gid,
			AdditionalGids: additionalGroups,
			// Username (Windows)
		},
		Args: args,
		Env:  env,
		Cwd:  cwd,
		Capabilities: &specs.LinuxCapabilities{
			Bounding:    bounding,
			Effective:   caps,
			Inheritable: bounding,
			Permitted:   bounding,
			Ambient:     ambient,
		},
		Rlimits:         rlimits,
		NoNewPrivileges: assignBool(label.NoNewPrivileges, yaml.NoNewPrivileges),
		// ApparmorProfile
		OOMScoreAdj: assignIntPtr(label.OOMScoreAdj, yaml.OOMScoreAdj),
		// SelinuxLabel
	}

	oci.Root = &specs.Root{
		Path:     "rootfs",
		Readonly: readonly,
	}

	oci.Hostname = assignStringEmpty(label.Hostname, yaml.Hostname)
	oci.Mounts = mountList

	resources := assignResources(label.Resources, yaml.Resources)

	oci.Linux = &specs.Linux{
		UIDMappings: assignMappings(label.UIDMappings, yaml.UIDMappings),
		GIDMappings: assignMappings(label.GIDMappings, yaml.GIDMappings),
		Sysctl:      assignMaps(label.Sysctl, yaml.Sysctl),
		Resources:   &resources,
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

	runtime = assignRuntime(label.Runtime, yaml.Runtime)

	return oci, runtime, nil
}
