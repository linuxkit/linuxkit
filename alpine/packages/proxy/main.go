package main

import (
	"log"
	"os"
	"pkg/proxy"
)

func main() {
	host, container := parseHostContainerAddrs()

	err := proxyForever(proxy.NewProxy(host, container))

	if err != nil {
		os.Exit(0)
	}
	os.Exit(1)
}
