package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
)

// NewHyperKitPlugin creates an instance plugin for hyperkit.
func NewHyperKitPlugin(vmLib, vmDir, hyperkit, vpnkitSock string) instance.Plugin {
	return &hyperkitPlugin{VMLib: vmLib,
		VMDir:      vmDir,
		HyperKit:   hyperkit,
		VPNKitSock: vpnkitSock}
}

type hyperkitPlugin struct {
	// VMLib is the path to a directory where each sub-directory
	// contains a vmlinuz and a initrd image
	VMLib string

	// VMDir is the path to a directory where per VM state is kept
	VMDir string

	// Hyperkit is the path to the hyperkit executable
	HyperKit string

	// VPNKitSock is the path to the VPNKit Unix domain socket.
	VPNKitSock string
}

const (
	hyperkitPid = "hyperkit.pid"
)

// Validate performs local validation on a provision request.
func (v hyperkitPlugin) Validate(req *types.Any) error {
	return nil
}

// Provision creates a new instance.
func (v hyperkitPlugin) Provision(spec instance.Spec) (*instance.ID, error) {

	var properties map[string]interface{}

	if spec.Properties != nil {
		if err := spec.Properties.Decode(&properties); err != nil {
			return nil, fmt.Errorf("Invalid instance properties: %s", err)
		}
	}

	if properties["Moby"] == nil {
		return nil, errors.New("Property 'Moby' must be set")
	}
	mobyStr, ok := properties["Moby"].(string)
	if !ok {
		return nil, errors.New("Property 'Moby' must be a string")
	}
	if properties["CPUs"] == nil {
		properties["CPUs"] = 1
	}
	numCPUs, ok := properties["CPUs"].(int)
	if !ok {
		return nil, errors.New("Property 'CPUs' must be a integer")
	}
	if properties["Memory"] == nil {
		properties["Memory"] = 512
	}
	memSz, ok := properties["Memory"].(int)
	if !ok {
		return nil, errors.New("Property 'Memory' must be a integer")
	}

	if properties["Disk"] == nil {
		properties["Disk"] = 256
	}
	diskSz, ok := properties["Disk"].(int)
	if !ok {
		return nil, errors.New("Property 'Disk' must be a integer")
	}

	instanceDir, err := ioutil.TempDir(v.VMDir, "infrakit-")
	if err != nil {
		return nil, err
	}

	id := instance.ID(path.Base(instanceDir))

	err = v.execHyperKit(instanceDir, mobyStr, numCPUs, memSz, diskSz)
	if err != nil {
		v.Destroy(id)
		return nil, err
	}

	tagData, err := types.AnyValue(spec.Tags)
	if err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(path.Join(instanceDir, "tags"), tagData.Bytes(), 0666); err != nil {
		return nil, err
	}

	return &id, nil
}

// Label labels the instance
func (v hyperkitPlugin) Label(instance instance.ID, labels map[string]string) error {
	instanceDir := path.Join(v.VMDir, string(instance))
	tagFile := path.Join(instanceDir, "tags")
	buff, err := ioutil.ReadFile(tagFile)
	if err != nil {
		return err
	}

	tags := map[string]string{}
	err = types.AnyBytes(buff).Decode(&tags)
	if err != nil {
		return err
	}

	for k, v := range labels {
		tags[k] = v
	}

	encoded, err := types.AnyValue(tags)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(tagFile, encoded.Bytes(), 0666)
}

// Destroy terminates an existing instance.
func (v hyperkitPlugin) Destroy(id instance.ID) error {
	fmt.Println("Destroying ", id)

	instanceDir := path.Join(v.VMDir, string(id))
	_, err := os.Stat(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("Instance does not exist")
		}
	}

	p, err := getProcess(instanceDir)
	if err != nil {
		log.Warningln("Can't find processes: ", err)
		return err
	}
	err = p.Kill()
	if err != nil {
		log.Warningln("Can't kill processes with pid: ", p.Pid, err)
		return err
	}

	if err := os.RemoveAll(instanceDir); err != nil {
		return err
	}

	return nil
}

// DescribeInstances returns descriptions of all instances matching all of the provided tags.
func (v hyperkitPlugin) DescribeInstances(tags map[string]string) ([]instance.Description, error) {
	files, err := ioutil.ReadDir(v.VMDir)
	if err != nil {
		return nil, err
	}

	descriptions := []instance.Description{}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		instanceDir := path.Join(v.VMDir, file.Name())

		tagData, err := ioutil.ReadFile(path.Join(instanceDir, "tags"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		instanceTags := map[string]string{}
		if err := types.AnyBytes(tagData).Decode(&instanceTags); err != nil {
			return nil, err
		}

		allMatched := true
		for k, v := range tags {
			value, exists := instanceTags[k]
			if !exists || v != value {
				allMatched = false
				break
			}
		}

		if allMatched {
			var logicalID *instance.LogicalID
			pidData, err := ioutil.ReadFile(path.Join(instanceDir, hyperkitPid))
			if err == nil {
				id := instance.LogicalID(pidData)
				logicalID = &id
			} else {
				if !os.IsNotExist(err) {
					return nil, err
				}
			}

			// Check if process is running
			if _, err := getProcess(instanceDir); err != nil {
				log.Warningln("Process not running: Instance ", file.Name())
				if err := os.RemoveAll(instanceDir); err != nil {
					log.Warningln("Can't remove instance dir ", instanceDir, " error ", err)

				}

				continue
			}

			descriptions = append(descriptions, instance.Description{
				ID:        instance.ID(file.Name()),
				LogicalID: logicalID,
				Tags:      instanceTags,
			})
		}
	}

	return descriptions, nil
}

func (v hyperkitPlugin) execHyperKit(instanceDir, moby string, cpus, memSz, diskSz int) error {
	err := createDisk(instanceDir, diskSz)
	if err != nil {
		return err
	}

	id := path.Base(instanceDir)

	c := []string{v.HyperKit, "-A", "-u"}

	// Pid file
	c = append(c, "-F", path.Join(instanceDir, hyperkitPid))

	// CPU and Memory
	c = append(c, "-c", fmt.Sprintf("%d", cpus))
	c = append(c, "-m", fmt.Sprintf("%dM", memSz))

	// Devices
	c = append(c, "-s", "0:0,hostbridge")
	c = append(c, "-s", "31,lpc")
	c = append(c, "-s", "5,virtio-rnd")
	c = append(c, "-s", fmt.Sprintf("4,virtio-blk,%s", path.Join(instanceDir, "disk.img")))
	c = append(c, "-s", fmt.Sprintf("2:0,virtio-vpnkit,path=%s", v.VPNKitSock))
	c = append(c, "-l", fmt.Sprintf("com1,autopty=%s,log=%s",
		path.Join(instanceDir, "tty"),
		path.Join(instanceDir, "console-ring")))

	// Kernel command line
	// Note, it is important that the kernel is one argv, not multiple
	kernStr := fmt.Sprintf("kexec,%s,%s,",
		path.Join(v.VMLib, moby, "vmlinuz64"),
		path.Join(v.VMLib, moby, "initrd.img"))
	kernStr += "earlyprintk=serial console=ttyS0"
	kernStr += " panic=1 vsyscall=emulate page_poison=1"
	kernStr += " ntp=gateway "
	c = append(c, "-f", kernStr)

	cmd := exec.Command(c[0], c[1:]...)
	cmd.Env = os.Environ()

	stdoutChan := make(chan string)
	stderrChan := make(chan string)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	stream(stdout, stdoutChan)
	stream(stderr, stderrChan)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case stderrl := <-stderrChan:
				log.Warningln("HyperKit ", id, " STDERR: ", stderrl)
			case stdoutl := <-stdoutChan:
				log.Infoln("HyperKit ", id, " STDOUT: ", stdoutl)
			case <-done:
				return
			}
		}
	}()

	log.Infoln("Starting ", id, ": ", c)

	err = cmd.Start()
	return err
}

func createDisk(instanceDir string, diskSz int) error {
	f, err := os.Create(path.Join(instanceDir, "disk.img"))
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 1048676)
	for i := 0; i < len(buf); i++ {
		buf[i] = 0
	}

	for i := 0; i < diskSz; i++ {
		f.Write(buf)
	}
	return nil
}

func stream(r io.ReadCloser, dest chan<- string) {
	go func() {
		defer r.Close()
		reader := bufio.NewReader(r)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			dest <- line
		}
	}()
}

func getProcess(instanceDir string) (*os.Process, error) {
	pidData, err := ioutil.ReadFile(path.Join(instanceDir, hyperkitPid))
	if err != nil {
		log.Warningln("Can't read pid file: ", err)
		return nil, err
	}
	pid, err := strconv.Atoi(string(pidData[:]))
	if err != nil {
		log.Warningln("Can't convert pidData: ", pidData, err)
		return nil, err
	}
	return os.FindProcess(pid)
}
