package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	stateDir      = "/state"
	exposePortCmd = "/usr/bin/vpnkit-expose-port"
)

// TODO cleanup logging, i.e. use debug log level for commands
// TODO implement updatePorts
// TODO emit Kubernetes events on errors (and perhaps success), so they can be observed by the user (via `kubectl get events`
//      or `kubectl describe service`), make sure errors user sees are meaningful
// TODO consider controller pod lifetime and lifetime of children, there are a few ways to go about it, do we really need to
//      write PID files? we want to persist them, we should mount state dir or use ports dir? or we could just use a channel,
//      if we expect children to die together with the parent controller...

func main() {
	flag.Parse()

	log.Println("Starting vpnkit-expose-port-controller...")

	if err := os.MkdirAll(stateDir, 0777); err != nil {
		log.Fatal(err)
	}

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatal(err)
	}

	restClient := clientset.Core().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "services", corev1.NamespaceAll, fields.Everything())

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(watchlist, &corev1.Service{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				err := openPorts(obj.(*corev1.Service), clientset)
				if err != nil {
					log.Println(err)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				err := updatePorts(oldObj.(*corev1.Service), newObj.(*corev1.Service))
				if err != nil {
					log.Println(err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				err := closePorts(obj.(*corev1.Service))
				if err != nil {
					log.Println(err)
				}
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutdown signal received, exiting...")
	close(stop)
}

func openPorts(service *corev1.Service, clientset *kubernetes.Clientset) error {
	for _, p := range service.Spec.Ports {
		if p.NodePort == 0 {
			continue
		}
		externalPort := fmt.Sprintf("%d", p.NodePort)
		proto := "tcp"
		if p.Protocol == corev1.ProtocolUDP {
			proto = "udp"
		}
		internalPort := fmt.Sprintf("%d", p.Port)
		args := []string{"-proto", proto, "-host-ip", "0.0.0.0", "-host-port", externalPort, "-container-ip", service.Spec.ClusterIP, "-container-port", internalPort, "-i", "-no-local-ip"}
		log.Printf("Calling \"%s %s\"", exposePortCmd, strings.Join(args, " "))
		cmd := exec.Command(exposePortCmd, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Start(); err != nil {
			return err
		}
		pid := cmd.Process.Pid
		go func() {
			if err := cmd.Wait(); err != nil {
				log.Printf("%s: %v", exposePortCmd, err)
				if out := stdout.String(); len(out) > 0 {
					log.Printf("stdout:\n%s", out)
				}
				if out := stderr.String(); len(out) > 0 {
					log.Printf("stderr:\n%s", out)
				}
				return
			}
			log.Printf("%s: pid %d exited", exposePortCmd, pid)
		}()
		// store the pid in the stateDir
		pidPath := fmt.Sprintf("%s/%s:%s:%s:%s.pid", stateDir, service.Namespace, service.Name, proto, externalPort)
		log.Printf("Started %s", pidPath)
		pidFile, err := os.Create(pidPath)
		if err != nil {
			return err
		}
		defer pidFile.Close()
		if _, err = pidFile.WriteString(fmt.Sprintf("%d", pid)); err != nil {
			return err
		}
	}
	return nil
}

func updatePorts(oldS, newS *corev1.Service) error {
	return errors.New("updatePorts not implemented")
}

func closePorts(service *corev1.Service) error {
	for _, p := range service.Spec.Ports {
		if p.NodePort == 0 {
			continue
		}
		externalPort := fmt.Sprintf("%d", p.NodePort)
		proto := "tcp"
		if p.Protocol == corev1.ProtocolUDP {
			proto = "udp"
		}
		pidPath := fmt.Sprintf("%s/%s:%s:%s:%s.pid", stateDir, service.Namespace, service.Name, proto, externalPort)
		b, err := ioutil.ReadFile(pidPath)
		if err != nil {
			return err
		}
		pid, err := strconv.ParseInt(string(b), 10, 64)
		if err != nil {
			return err
		}
		cmd, _ := os.FindProcess(int(pid)) // always succeeds on Unix
		if err = cmd.Signal(os.Kill); err != nil {
			log.Printf("Failed to kill %s with pid %d: %v", pidPath, int(pid), err)
		} else {
			log.Printf("Killed %s with pid %d", pidPath, int(pid))
		}
		if err := os.Remove(pidPath); err != nil {
			return err
		}
	}
	return nil
}
