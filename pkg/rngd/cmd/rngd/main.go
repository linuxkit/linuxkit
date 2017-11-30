// +build linux

package main

// int rndaddentropy;
import "C"

import (
	"encoding/binary"
	"errors"
	"flag"
	"log"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

type rng struct {
	rngName  string
	rngDev   string
	disabled bool
	rngInit  xinit
	rngRead  xread
}

type xinit func(ctx *rng) bool
type xread func(ctx *rng) (uint64, error)

// Check if hardware RNG device is valid
func initEntropySource(ctx *rng) bool {
	f, err := os.Open(ctx.rngDev)
	defer f.Close()
	if err != nil {
		ctx.disabled = true
		return false
	}

	// Try to read some data from the entropy source. If it doesn't
	// return an error, assume it's ok to use
	buf := make([]byte, 16)
	n, err := f.Read(buf)
	if err != nil || n != 16 {
		ctx.disabled = true
		return false
	}

	return true
}

// Read entropy from the hardware RNG device
func readHwRNG(ctx *rng) (uint64, error) {
	f, err := os.Open(ctx.rngDev)
	defer f.Close()
	if err != nil {
		return 0, err
	}
	buf := make([]byte, 8)
	_, err = f.Read(buf)
	if err != nil {
		return 0, err
	}

	r := binary.LittleEndian.Uint64(buf)

	return r, nil
}

var entropySources = []rng{
	// Hardware RNG device
	{
		rngName: "Hardware RNG Device",
		rngDev:  "/dev/hwrng",
		rngInit: initEntropySource,
		rngRead: readHwRNG,
	},
	// RNG instruction support
	{
		rngName: "Instruction RNG",
		rngInit: initDRNG, // Arch specific
		rngRead: readDRNG, // Arch specific
	},
}

func main() {
	var entSource bool
	oneshot := flag.Bool("1", false, "Enable oneshot mode")
	flag.Parse()

	timeout := -1
	if *oneshot {
		timeout = 0
	}

	for _, e := range entropySources {
		if e.rngInit != nil && e.rngInit(&e) {
			entSource = true
		}
	}
	if entSource == false {
		log.Fatalf("No random source available")
	}

	random, err := os.Open("/dev/random")
	if err != nil {
		log.Fatalf("Cannot open /dev/random: %v", err)
	}
	defer random.Close()
	fd := int(random.Fd())

	epfd, err := unix.EpollCreate1(0)
	if err != nil {
		log.Fatalf("epoll create error: %v", err)
	}
	defer unix.Close(epfd)

	var event unix.EpollEvent
	var events [1]unix.EpollEvent

	event.Events = unix.EPOLLOUT
	event.Fd = int32(fd)
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		log.Fatalf("epoll add error: %v", err)
	}

	count := 0

	for {
		// write some entropy
		n, err := writeEntropy(random)
		if err != nil {
			log.Fatalf("write entropy: %v", err)
		}
		count += n
		// sleep until we can write more
		nevents, err := unix.EpollWait(epfd, events[:], timeout)
		if err != nil {
			log.Fatalf("epoll wait error: %v", err)
		}
		if nevents == 1 && events[0].Events&unix.EPOLLOUT == unix.EPOLLOUT {
			continue
		}
		if *oneshot {
			log.Printf("Wrote %d bytes of entropy, exiting as oneshot\n", count)
			break
		}
	}
}

type randInfo struct {
	entropyCount int
	size         int
	buf          uint64
}

func writeEntropy(random *os.File) (int, error) {
	var i int
	var r uint64
	var err error
	var ret uintptr
	for _, e := range entropySources {
		if e.disabled {
			continue
		}
		if e.rngRead != nil {
			r, err = e.rngRead(&e)
			if err != nil {
				continue
			}
		}

		const entropy = 64 // they are good random numbers, Brent
		info := randInfo{entropy, 8, r}
		ret, _, _ = unix.Syscall(unix.SYS_IOCTL, uintptr(random.Fd()), uintptr(C.rndaddentropy), uintptr(unsafe.Pointer(&info)))
		if ret == 0 {
			i += 8
		} else {
			continue
		}
	}
	if i == 0 {
		return 0, errors.New("No entropy added")
	}

	return i, nil
}
