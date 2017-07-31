package main

import (
	"log"
	"os"
	"syscall"
)

func main() {
	oneshot := len(os.Args) > 1 && os.Args[1] == "-1"

	timeout := -1
	if oneshot {
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

	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		log.Fatalf("epoll create error: %v", err)
	}
	defer syscall.Close(epfd)

	var event syscall.EpollEvent
	var events [1]syscall.EpollEvent

	event.Events = syscall.EPOLLOUT
	event.Fd = int32(fd)
	if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, fd, &event); err != nil {
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
		nevents, err := syscall.EpollWait(epfd, events[:], timeout)
		if err != nil {
			log.Fatalf("epoll wait error: %v", err)
		}
		if nevents == 1 && events[0].Events&syscall.EPOLLOUT == syscall.EPOLLOUT {
			continue
		}
		if oneshot {
			log.Printf("Wrote %d bytes of entropy, exiting as oneshot\n", count)
			break
		}
	}
}
