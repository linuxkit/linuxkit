// +build linux

package main

import (
	"flag"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

func main() {
	oneshot := flag.Bool("1", false, "Enable oneshot mode")
	flag.Parse()

	timeout := -1
	if *oneshot {
		timeout = 0
	}

	supported := initRand()
	if !supported {
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
