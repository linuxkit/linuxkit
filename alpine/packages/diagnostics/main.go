package main

import (
	"flag"
	"log"
	"log/syslog"
)

func init() {
	syslog, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "diagnostics")
	if err != nil {
		log.Fatalln("Failed to open syslog", err)
	}

	log.SetOutput(syslog)
	log.SetFlags(0)
}

// DiagnosticListener listens for starting diagnostics capture requests
type DiagnosticListener interface {
	// Listen(), a blocking method intended to be invoked in its own
	// goroutine, will listen for a diagnostic information request and
	// capture the desired information if one is received.
	Listen()
}

// DiagnosticUploader uploads the collected information to the mothership.
type DiagnosticUploader interface {
	Upload() error
}

func main() {
	flHTTP := flag.Bool("http", false, "Enable diagnostic HTTP listener")
	flVSock := flag.Bool("vsock", false, "Enable vsock listener")
	flHVSock := flag.Bool("hvsock", false, "Enable hvsock listener")
	flRawTCP := flag.Bool("rawtcp", false, "Enable raw TCP listener")

	flag.Parse()

	listeners := make([]DiagnosticListener, 0)

	if *flHTTP {
		listeners = append(listeners, HTTPDiagnosticListener{})
	}

	if *flVSock {
		listeners = append(listeners, VSockDiagnosticListener{})
	}

	if *flHVSock {
		listeners = append(listeners, HVSockDiagnosticListener{})
	}

	if *flRawTCP {
		listeners = append(listeners, RawTCPDiagnosticListener{})
	}

	for _, l := range listeners {
		go l.Listen()
	}
	forever := make(chan int)
	<-forever
}
