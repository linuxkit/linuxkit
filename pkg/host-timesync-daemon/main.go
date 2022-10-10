package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/linuxkit/virtsock/pkg/vsock"
)

// Listen for connections on an AF_VSOCK address and update the system time
// from the hardware clock when a connection is received.

// From <linux/rtc.h> struct rtc_time
type rtcTime struct {
	tmSec   uint32
	tmMin   uint32
	tmHour  uint32
	tmMday  uint32
	tmMon   uint32
	tmYear  uint32
	tmWday  uint32
	tmYday  uint32
	tmIsdst uint32
}

const (
	// iocREAD and friends are from <linux/asm-generic/ioctl.h>
	iocREAD      = uintptr(2)
	iocNRBITS    = uintptr(8)
	iocNRSHIFT   = uintptr(0)
	iocTYPEBITS  = uintptr(8)
	iocTYPESHIFT = iocNRSHIFT + iocNRBITS
	iocSIZEBITS  = uintptr(14)
	iocSIZESHIFT = iocTYPESHIFT + iocTYPEBITS
	iocDIRSHIFT  = iocSIZESHIFT + iocSIZEBITS
	// rtcRDTIMENR and friends are from <linux/rtc.h>
	rtcRDTIMENR   = uintptr(0x09)
	rtcRDTIMETYPE = uintptr(112)
)

func rtcReadTime() rtcTime {
	f, err := os.Open("/dev/rtc0")
	if err != nil {
		log.Fatalf("Failed to open /dev/rtc0: %v", err)
	}
	defer f.Close()
	result := rtcTime{}
	arg := uintptr(0)
	arg |= (iocREAD << iocDIRSHIFT)
	arg |= (rtcRDTIMETYPE << iocTYPESHIFT)
	arg |= (rtcRDTIMENR << iocNRSHIFT)
	arg |= (unsafe.Sizeof(result) << iocSIZESHIFT)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), arg, uintptr(unsafe.Pointer(&result)))
	if errno != 0 {
		log.Fatalf("RTC_RD_TIME failed: %v", errno)
	}
	return result
}

func main() {
	// host-timesync-daemon -cid <cid> -port <port>

	cid := flag.Int("cid", 0, "AF_VSOCK CID to listen on")
	port := flag.Int("port", 0, "AF_VSOCK port to listen on")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s: set the time after an AH_VSOCK connection is received.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example usage:\n")
		fmt.Fprintf(os.Stderr, "%s -port 0xf3a4\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   -- when a connection is received on port 0xf3a4, query the hardware clock and\n")
		fmt.Fprintf(os.Stderr, "      set the system time. The connection will be closed after the clock has\n")
		fmt.Fprintf(os.Stderr, "      been changed.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	if *port == 0 {
		log.Fatalf("Please supply a -port argument")
	}
	if *cid == 0 {
		// by default allow connections from anywhere on the local machine
		*cid = vsock.CIDAny
	}

	l, err := vsock.Listen(uint32(*cid), uint32(*port))
	if err != nil {
		log.Fatalf("Failed to bind to vsock port %x:%x: %s", *cid, *port, err)
	}
	log.Printf("Listening on port %x:%x", *cid, *port)

	for {
		conn, err := l.Accept()

		if err != nil {
			log.Fatalf("Error accepting connection: %s", err)
		}
		log.Printf("Connection to: %x:%x from: %s\n", *cid, *port, conn.RemoteAddr())

		t := rtcReadTime()
		// Assume the RTC is set to UTC. This may not be true on a Windows host but in
		// that case we assume the platform is capable of providing a PV clock and
		// we don't use this code anyway.
		d := time.Date(int(t.tmYear+1900), time.Month(t.tmMon+1), int(t.tmMday), int(t.tmHour), int(t.tmMin), int(t.tmSec), 0, time.UTC)
		log.Printf("Setting system clock to %s", d)
		tv := syscall.Timeval{
			Sec:  d.Unix(),
			Usec: 0, // the RTC only has second granularity
		}
		if err = syscall.Settimeofday(&tv); err != nil {
			log.Printf("Unexpected failure from Settimeofday: %v", err)
		}
		// Close after the command terminates. The caller can use this as a notification.
		conn.Close()
	}
}
