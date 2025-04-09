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

	"github.com/equinix/equinix-sdk-go/services/metalv1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

const (
	equinixmetalDefaultZone    = "ams1"
	equinixmetalDefaultMachine = "baremetal_0"
	equinixmetalBaseURL        = "METAL_BASE_URL"
	equinixmetalZoneVar        = "METAL_FACILITY"
	equinixmetalMachineVar     = "METAL_MACHINE"
	equinixmetalAPIKeyVar      = "METAL_API_TOKEN"
	equinixmetalProjectIDVar   = "METAL_PROJECT_ID"
	equinixmetalHostnameVar    = "METAL_HOSTNAME"
	equinixmetalNameVar        = "METAL_NAME"
)

var (
	equinixmetalDefaultHostname = "linuxkit"
)

func init() {
	// Prefix host name with username
	if u, err := user.Current(); err == nil {
		equinixmetalDefaultHostname = u.Username + "-" + equinixmetalDefaultHostname
	}
}

func runEquinixMetalCmd() *cobra.Command {
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
		Use:   "equinixmetal",
		Short: "launch an Equinix Metal device",
		Long: `Launch an Equinix Metal device.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run equinixmetal [options] name",
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := "equinixmetal"
			if len(args) > 0 {
				prefix = args[0]
			}
			url := getStringValue(equinixmetalBaseURL, baseURLFlag, "")
			if url == "" {
				return fmt.Errorf("need to specify a value for --base-url where the images are hosted. This URL should contain <url>/%s-kernel, <url>/%s-initrd.img and <url>/%s-equinixmetal.ipxe", prefix, prefix, prefix)
			}
			facility := getStringValue(equinixmetalZoneVar, zoneFlag, "")
			plan := getStringValue(equinixmetalMachineVar, machineFlag, defaultMachine)
			apiKey := getStringValue(equinixmetalAPIKeyVar, apiKeyFlag, "")
			if apiKey == "" {
				return errors.New("must specify an api.equinix.com API key with --api-key")
			}
			projectID := getStringValue(equinixmetalProjectIDVar, projectFlag, "")
			if projectID == "" {
				return errors.New("must specify an api.equinix.com Project ID with --project-id")
			}
			hostname := getStringValue(equinixmetalHostnameVar, hostNameFlag, "")
			name := getStringValue(equinixmetalNameVar, nameFlag, prefix)
			osType := "custom_ipxe"
			billing := "hourly"

			if !keepFlag && !consoleFlag {
				return fmt.Errorf("combination of keep=%t and console=%t makes little sense", keepFlag, consoleFlag)
			}

			ipxeScriptName := fmt.Sprintf("%s-equinixmetal.ipxe", name)

			// Serve files with a local http server
			var httpServer *http.Server
			if serveFlag != "" {
				// Read kernel command line
				var cmdline string
				if c, err := os.ReadFile(prefix + "-cmdline"); err != nil {
					return fmt.Errorf("cannot open cmdline file: %v", err)
				} else {
					cmdline = string(c)
				}

				ipxeScript := equinixmetalIPXEScript(name, url, cmdline, equinixmetalMachineToArch(machineFlag))
				log.Debugf("Using iPXE script:\n%s\n", ipxeScript)

				// Two handlers, one for the iPXE script and one for the kernel/initrd files
				mux := http.NewServeMux()
				mux.HandleFunc(fmt.Sprintf("/%s", ipxeScriptName),
					func(w http.ResponseWriter, r *http.Request) {
						_, _ = fmt.Fprint(w, ipxeScript)
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
				return fmt.Errorf("invalid iPXE URL %s: %v", ipxeURL, err)
			}
			log.Infof("Validating URL: %s", kernelURL)
			if err := validateHTTPURL(kernelURL); err != nil {
				return fmt.Errorf("invalid kernel URL %s: %v", kernelURL, err)
			}
			log.Infof("Validating URL: %s", initrdURL)
			if err := validateHTTPURL(initrdURL); err != nil {
				return fmt.Errorf("invalid initrd URL %s: %v", initrdURL, err)
			}

			client := metalv1.NewAPIClient(&metalv1.Configuration{})
			metalCtx := context.WithValue(
				context.Background(),
				metalv1.ContextAPIKeys,
				map[string]metalv1.APIKey{
					"X-Auth-Token": {Key: apiKey},
				},
			)
			var tags []string

			var dev *metalv1.Device
			var err error
			if deviceFlag != "" {
				dev, _, err = client.DevicesApi.FindDeviceByIdExecute(client.DevicesApi.FindDeviceById(metalCtx, deviceFlag))
				if err != nil {
					return fmt.Errorf("getting info for device %s failed: %v", deviceFlag, err)
				}
				b, err := json.MarshalIndent(dev, "", "    ")
				if err != nil {
					log.Fatal(err)
				}
				log.Debugf("%s\n", string(b))

				updateReq := client.DevicesApi.UpdateDevice(metalCtx, deviceFlag)
				updateReq.DeviceUpdateInput(metalv1.DeviceUpdateInput{
					Hostname:      &hostname,
					Locked:        dev.Locked,
					Tags:          dev.Tags,
					IpxeScriptUrl: &ipxeURL,
					AlwaysPxe:     &alwaysPXE,
				})
				dev, _, err = client.DevicesApi.UpdateDeviceExecute(updateReq)
				if err != nil {
					return fmt.Errorf("update device %s failed: %v", deviceFlag, err)
				}

				actionReq := client.DevicesApi.PerformAction(metalCtx, deviceFlag)
				actionReq.DeviceActionInput(metalv1.DeviceActionInput{Type: metalv1.DEVICEACTIONINPUTTYPE_REBOOT})
				if _, err := client.DevicesApi.PerformActionExecute(actionReq); err != nil {
					return fmt.Errorf("rebooting device %s failed: %v", deviceFlag, err)
				}
			} else {
				// Create a new device
				createReq := client.DevicesApi.CreateDevice(metalCtx, projectID)
				billingCycle := metalv1.DeviceCreateInputBillingCycle(billing)
				createReq.CreateDeviceRequest(metalv1.CreateDeviceRequest{
					DeviceCreateInFacilityInput: &metalv1.DeviceCreateInFacilityInput{
						Hostname:        &hostname,
						Plan:            plan,
						Facility:        []string{facility},
						OperatingSystem: osType,
						BillingCycle:    &billingCycle,
						Tags:            tags,
						IpxeScriptUrl:   &ipxeURL,
						AlwaysPxe:       &alwaysPXE,
					},
				})
				dev, _, err = client.DevicesApi.CreateDeviceExecute(createReq)
				if err != nil {
					return fmt.Errorf("creating device failed: %w", err)
				}
			}
			b, err := json.MarshalIndent(dev, "", "    ")
			if err != nil {
				return err
			}
			log.Debugf("%s\n", string(b))

			log.Printf("Booting %s...", *dev.Id)

			sshHost := "sos." + *dev.Facility.Code + ".platformequinix.com"
			if consoleFlag {
				// Connect to the serial console
				if err := equinixmetalSOS(*dev.Id, sshHost); err != nil {
					return err
				}
			} else {
				log.Printf("Access the console with: ssh %s@%s", *dev.Id, sshHost)

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
				log.Printf("Device ID: %s", *dev.Id)
				log.Printf("Serial:    ssh %s@%s", *dev.Id, sshHost)
			} else {
				deleteReq := client.DevicesApi.DeleteDevice(metalCtx, *dev.Id)
				if _, err := client.DevicesApi.DeleteDeviceExecute(deleteReq); err != nil {
					return fmt.Errorf("unable to delete device: %v", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURLFlag, "base-url", "", "Base URL that the kernel, initrd and iPXE script are served from (or "+equinixmetalBaseURL+")")
	cmd.Flags().StringVar(&zoneFlag, "zone", equinixmetalDefaultZone, "Equinix Metal Facility (or "+equinixmetalZoneVar+")")
	cmd.Flags().StringVar(&machineFlag, "machine", equinixmetalDefaultMachine, "Equinix Metal Machine Type (or "+equinixmetalMachineVar+")")
	cmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "Equinix Metal API key (or "+equinixmetalAPIKeyVar+")")
	cmd.Flags().StringVar(&projectFlag, "project-id", "", "EquinixMetal Project ID (or "+equinixmetalProjectIDVar+")")
	cmd.Flags().StringVar(&deviceFlag, "device", "", "The ID of an existing device")
	cmd.Flags().StringVar(&hostNameFlag, "hostname", equinixmetalDefaultHostname, "Hostname of new instance (or "+equinixmetalHostnameVar+")")
	cmd.Flags().StringVar(&nameFlag, "img-name", "", "Overrides the prefix used to identify the files. Defaults to [name] (or "+equinixmetalNameVar+")")
	cmd.Flags().BoolVar(&alwaysPXE, "always-pxe", true, "Reboot from PXE every time.")
	cmd.Flags().StringVar(&serveFlag, "serve", "", "Serve local files via the http port specified, e.g. ':8080'.")
	cmd.Flags().BoolVar(&consoleFlag, "console", true, "Provide interactive access on the console.")
	cmd.Flags().BoolVar(&keepFlag, "keep", false, "Keep the machine after exiting/poweroff.")

	return cmd
}

// Convert machine type to architecture
func equinixmetalMachineToArch(machine string) string {
	switch machine {
	case "baremetal_2a", "baremetal_2a2":
		return "aarch64"
	default:
		return "x86_64"
	}
}

// Build the iPXE script for equinix metal machines
func equinixmetalIPXEScript(name, baseURL, cmdline, arch string) string {
	// Note, we *append* the <prefix>-cmdline. iXPE booting will
	// need the first set of "kernel-params" and we don't want to
	// require these to be added to every YAML file.
	script := "#!ipxe\n\n"
	script += "dhcp\n"
	script += fmt.Sprintf("set base-url %s\n", baseURL)
	if arch != "aarch64" {
		var tty string
		// x86_64 Equinix Metal machines have console on non standard ttyS1 which is not in most examples
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
		return fmt.Errorf("got a non 200- or 300- HTTP response code: %s", resp.Status)
	}
	return nil
}

func equinixmetalSOS(user, host string) error {
	log.Debugf("console: ssh %s@%s", user, host)

	hostKey, err := sshHostKey(host)
	if err != nil {
		return fmt.Errorf("host key not found. Maybe need to add it? %v", err)
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
		return fmt.Errorf("failed to dial: %s", err)
	}

	s, err := c.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = s.Close()
	}()

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
		return fmt.Errorf("request for PTY failed: %v", err)
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
		return fmt.Errorf("failed to start shell: %v", err)
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
		return nil, fmt.Errorf("can't read known_hosts file: %v", err)
	}

	for {
		marker, hosts, pubKey, _, rest, err := ssh.ParseKnownHosts(f)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse error in known_hosts: %v", err)
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

	return nil, fmt.Errorf("no hostkey for %s", host)
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
	return nil, fmt.Errorf("file %s not found", name)
}
