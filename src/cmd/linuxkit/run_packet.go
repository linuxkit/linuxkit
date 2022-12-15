package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/packethost/packngo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

const (
	packetDefaultZone    = "ams1"
	packetDefaultMachine = "baremetal_0"
	packetBaseURL        = "PACKET_BASE_URL"
	packetZoneVar        = "PACKET_ZONE"
	packetMachineVar     = "PACKET_MACHINE"
	packetAPIKeyVar      = "PACKET_API_KEY"
	packetProjectIDVar   = "PACKET_PROJECT_ID"
	packetHostnameVar    = "PACKET_HOSTNAME"
	packetNameVar        = "PACKET_NAME"
)

var (
	packetDefaultHostname = "linuxkit"
)

func init() {
	// Prefix host name with username
	if u, err := user.Current(); err == nil {
		packetDefaultHostname = u.Username + "-" + packetDefaultHostname
	}
}

func runPacketCmd() *cobra.Command {
	var (
		baseURLFlag  string
		zoneFlag     string
		machineFlag  string
		apiKeyFlag   string
		projectFlag  string
		deviceFlag   string
		hostNameFlag string
		nameFlag     string
		alwaysPXE    bool
		serveFlag    string
		consoleFlag  bool
		keepFlag     bool
	)

	cmd := &cobra.Command{
		Use:   "packet",
		Short: "launch an Equinix Metal (Packet) device",
		Long: `Launch an Equinix Metal (Packet) device.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run packet [options] name",
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := "packet"
			if len(args) > 0 {
				prefix = args[0]
			}
			url := getStringValue(packetBaseURL, baseURLFlag, "")
			if url == "" {
				return fmt.Errorf("Need to specify a value for --base-url where the images are hosted. This URL should contain <url>/%s-kernel, <url>/%s-initrd.img and <url>/%s-packet.ipxe", prefix, prefix, prefix)
			}
			facility := getStringValue(packetZoneVar, zoneFlag, "")
			plan := getStringValue(packetMachineVar, machineFlag, defaultMachine)
			apiKey := getStringValue(packetAPIKeyVar, apiKeyFlag, "")
			if apiKey == "" {
				return errors.New("Must specify a Packet.net API key with --api-key")
			}
			projectID := getStringValue(packetProjectIDVar, projectFlag, "")
			if projectID == "" {
				return errors.New("Must specify a Packet.net Project ID with --project-id")
			}
			hostname := getStringValue(packetHostnameVar, hostNameFlag, "")
			name := getStringValue(packetNameVar, nameFlag, prefix)
			osType := "custom_ipxe"
			billing := "hourly"

			if !keepFlag && !consoleFlag {
				return fmt.Errorf("Combination of keep=%t and console=%t makes little sense", keepFlag, consoleFlag)
			}

			ipxeScriptName := fmt.Sprintf("%s-packet.ipxe", name)

			// Serve files with a local http server
			var httpServer *http.Server
			if serveFlag != "" {
				// Read kernel command line
				var cmdline string
				if c, err := os.ReadFile(prefix + "-cmdline"); err != nil {
					return fmt.Errorf("Cannot open cmdline file: %v", err)
				} else {
					cmdline = string(c)
				}

				ipxeScript := packetIPXEScript(name, url, cmdline, packetMachineToArch(machineFlag))
				log.Debugf("Using iPXE script:\n%s\n", ipxeScript)

				// Two handlers, one for the iPXE script and one for the kernel/initrd files
				mux := http.NewServeMux()
				mux.HandleFunc(fmt.Sprintf("/%s", ipxeScriptName),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprint(w, ipxeScript)
					})
				fs := serveFiles{[]string{fmt.Sprintf("%s-kernel", name), fmt.Sprintf("%s-initrd.img", name)}}
				mux.Handle("/", http.FileServer(fs))
				httpServer = &http.Server{Addr: serveFlag, Handler: mux}
				go func() {
					log.Debugf("Listening on http://%s\n", serveFlag)
					if err := httpServer.ListenAndServe(); err != nil {
						log.Infof("http server exited with: %v", err)
					}
				}()
			}

			// Make sure the URLs work
			ipxeURL := fmt.Sprintf("%s/%s", url, ipxeScriptName)
			initrdURL := fmt.Sprintf("%s/%s-initrd.img", url, name)
			kernelURL := fmt.Sprintf("%s/%s-kernel", url, name)
			log.Infof("Validating URL: %s", ipxeURL)
			if err := validateHTTPURL(ipxeURL); err != nil {
				return fmt.Errorf("Invalid iPXE URL %s: %v", ipxeURL, err)
			}
			log.Infof("Validating URL: %s", kernelURL)
			if err := validateHTTPURL(kernelURL); err != nil {
				return fmt.Errorf("Invalid kernel URL %s: %v", kernelURL, err)
			}
			log.Infof("Validating URL: %s", initrdURL)
			if err := validateHTTPURL(initrdURL); err != nil {
				return fmt.Errorf("Invalid initrd URL %s: %v", initrdURL, err)
			}

			client := packngo.NewClient("", apiKey, nil)
			var tags []string

			var dev *packngo.Device
			var err error
			if deviceFlag != "" {
				dev, _, err = client.Devices.Get(deviceFlag)
				if err != nil {
					return fmt.Errorf("Getting info for device %s failed: %v", deviceFlag, err)
				}
				b, err := json.MarshalIndent(dev, "", "    ")
				if err != nil {
					log.Fatal(err)
				}
				log.Debugf("%s\n", string(b))

				req := packngo.DeviceUpdateRequest{
					Hostname:      hostname,
					Locked:        dev.Locked,
					Tags:          dev.Tags,
					IPXEScriptURL: ipxeURL,
					AlwaysPXE:     alwaysPXE,
				}
				dev, _, err = client.Devices.Update(deviceFlag, &req)
				if err != nil {
					return fmt.Errorf("Update device %s failed: %v", deviceFlag, err)
				}
				if _, err := client.Devices.Reboot(deviceFlag); err != nil {
					return fmt.Errorf("Rebooting device %s failed: %v", deviceFlag, err)
				}
			} else {
				// Create a new device
				req := packngo.DeviceCreateRequest{
					Hostname:      hostname,
					Plan:          plan,
					Facility:      facility,
					OS:            osType,
					BillingCycle:  billing,
					ProjectID:     projectID,
					Tags:          tags,
					IPXEScriptURL: ipxeURL,
					AlwaysPXE:     alwaysPXE,
				}
				dev, _, err = client.Devices.Create(&req)
				if err != nil {
					return fmt.Errorf("Creating device failed: %w", err)
				}
			}
			b, err := json.MarshalIndent(dev, "", "    ")
			if err != nil {
				return err
			}
			log.Debugf("%s\n", string(b))

			log.Printf("Booting %s...", dev.ID)

			sshHost := "sos." + dev.Facility.Code + ".packet.net"
			if consoleFlag {
				// Connect to the serial console
				if err := packetSOS(dev.ID, sshHost); err != nil {
					return err
				}
			} else {
				log.Printf("Access the console with: ssh %s@%s", dev.ID, sshHost)

				// if the serve option is present, wait till 'ctrl-c' is hit.
				// Otherwise we wouldn't serve the files
				if serveFlag != "" {
					stop := make(chan os.Signal, 1)
					signal.Notify(stop, os.Interrupt)
					log.Printf("Hit ctrl-c to stop http server")
					<-stop
				}
			}

			// Stop the http server before exiting
			if serveFlag != "" {
				log.Debugf("Shutting down http server...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = httpServer.Shutdown(ctx)
			}

			if keepFlag {
				log.Printf("The machine is kept...")
				log.Printf("Device ID: %s", dev.ID)
				log.Printf("Serial:    ssh %s@%s", dev.ID, sshHost)
			} else {
				if _, err := client.Devices.Delete(dev.ID); err != nil {
					return fmt.Errorf("Unable to delete device: %v", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURLFlag, "base-url", "", "Base URL that the kernel, initrd and iPXE script are served from (or "+packetBaseURL+")")
	cmd.Flags().StringVar(&zoneFlag, "zone", packetDefaultZone, "Packet Zone (or "+packetZoneVar+")")
	cmd.Flags().StringVar(&machineFlag, "machine", packetDefaultMachine, "Packet Machine Type (or "+packetMachineVar+")")
	cmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "Packet API key (or "+packetAPIKeyVar+")")
	cmd.Flags().StringVar(&projectFlag, "project-id", "", "Packet Project ID (or "+packetProjectIDVar+")")
	cmd.Flags().StringVar(&deviceFlag, "device", "", "The ID of an existing device")
	cmd.Flags().StringVar(&hostNameFlag, "hostname", packetDefaultHostname, "Hostname of new instance (or "+packetHostnameVar+")")
	cmd.Flags().StringVar(&nameFlag, "img-name", "", "Overrides the prefix used to identify the files. Defaults to [name] (or "+packetNameVar+")")
	cmd.Flags().BoolVar(&alwaysPXE, "always-pxe", true, "Reboot from PXE every time.")
	cmd.Flags().StringVar(&serveFlag, "serve", "", "Serve local files via the http port specified, e.g. ':8080'.")
	cmd.Flags().BoolVar(&consoleFlag, "console", true, "Provide interactive access on the console.")
	cmd.Flags().BoolVar(&keepFlag, "keep", false, "Keep the machine after exiting/poweroff.")

	return cmd
}

// Convert machine type to architecture
func packetMachineToArch(machine string) string {
	switch machine {
	case "baremetal_2a", "baremetal_2a2":
		return "aarch64"
	default:
		return "x86_64"
	}
}

// Build the iPXE script for packet machines
func packetIPXEScript(name, baseURL, cmdline, arch string) string {
	// Note, we *append* the <prefix>-cmdline. iXPE booting will
	// need the first set of "kernel-params" and we don't want to
	// require these to be added to every YAML file.
	script := "#!ipxe\n\n"
	script += "dhcp\n"
	script += fmt.Sprintf("set base-url %s\n", baseURL)
	if arch != "aarch64" {
		var tty string
		// x86_64 Packet machines have console on non standard ttyS1 which is not in most examples
		if !strings.Contains(cmdline, "console=ttyS1") {
			tty = "console=ttyS1,115200"
		}
		script += fmt.Sprintf("set kernel-params ip=dhcp nomodeset ro serial %s %s\n", tty, cmdline)
		script += fmt.Sprintf("kernel ${base-url}/%s-kernel ${kernel-params}\n", name)
		script += fmt.Sprintf("initrd ${base-url}/%s-initrd.img\n", name)
	} else {
		// With EFI boot need to specify the initrd and root dev explicitly. See:
		// http://ipxe.org/appnote/debian_preseed
		// http://forum.ipxe.org/showthread.php?tid=7589
		script += fmt.Sprintf("initrd --name initrd ${base-url}/%s-initrd.img\n", name)
		script += fmt.Sprintf("set kernel-params ip=dhcp nomodeset ro %s\n", cmdline)
		script += fmt.Sprintf("kernel ${base-url}/%s-kernel initrd=initrd root=/dev/ram0 ${kernel-params}\n", name)
	}
	script += "boot"
	return script
}

// validateHTTPURL does a sanity check that a URL returns a 200 or 300 response
func validateHTTPURL(url string) error {
	resp, err := http.Head(url)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Got a non 200- or 300- HTTP response code: %s", resp.Status)
	}
	return nil
}

func packetSOS(user, host string) error {
	log.Debugf("console: ssh %s@%s", user, host)

	hostKey, err := sshHostKey(host)
	if err != nil {
		return fmt.Errorf("Host key not found. Maybe need to add it? %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.FixedHostKey(hostKey),
		Auth: []ssh.AuthMethod{
			sshAgent(),
		},
	}

	c, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}

	s, err := c.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %v", err)
	}
	defer s.Close()

	s.Stdout = os.Stdout
	s.Stderr = os.Stderr
	s.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:  0,
		ssh.IGNCR: 1,
	}

	width, height, err := term.GetSize(0)
	if err != nil {
		log.Warningf("Error getting terminal size. Ignored. %v", err)
		width = 80
		height = 40
	}
	if err := s.RequestPty("vt100", width, height, modes); err != nil {
		return fmt.Errorf("Request for PTY failed: %v", err)
	}
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() {
		_ = term.Restore(0, oldState)
	}()

	// Start remote shell
	if err := s.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %v", err)
	}

	_ = s.Wait()
	return nil
}

// Get a ssh-agent AuthMethod
func sshAgent() ssh.AuthMethod {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatalf("Failed to dial ssh-agent: %v", err)
	}
	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
}

// This function returns the host key for a given host (the SOS server).
// If it can't be found, it errors
func sshHostKey(host string) (ssh.PublicKey, error) {
	f, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return nil, fmt.Errorf("Can't read known_hosts file: %v", err)
	}

	for {
		marker, hosts, pubKey, _, rest, err := ssh.ParseKnownHosts(f)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Parse error in known_hosts: %v", err)
		}
		if marker != "" {
			//ignore CA or revoked key
			fmt.Printf("ignoring marker: %s\n", marker)
			continue
		}
		for _, h := range hosts {
			if h == host {
				return pubKey, nil
			}
		}
		f = rest
	}

	return nil, fmt.Errorf("No hostkey for %s", host)
}

// This implements a http.FileSystem which only responds to specific files.
type serveFiles struct {
	files []string
}

// Open implements the Open method for the serveFiles FileSystem
// implementation.
// It converts both the name from the URL and the files provided in
// the serveFiles structure into cleaned, absolute filesystem path and
// only returns the file if the requested name matches one of the
// files in the list.
func (fs serveFiles) Open(name string) (http.File, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	name = filepath.Join(cwd, filepath.FromSlash(path.Clean("/"+name)))
	for _, fn := range fs.files {
		fn = filepath.Join(cwd, filepath.FromSlash(path.Clean("/"+fn)))
		if name == fn {
			f, err := os.Open(fn)
			if err != nil {
				return nil, err
			}
			log.Debugf("Serving: %s", fn)
			return f, nil
		}
	}
	return nil, fmt.Errorf("File %s not found", name)
}
