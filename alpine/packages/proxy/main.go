package main

import (
	"pkg/proxy"
)

func main() {
	host, container := parseHostContainerAddrs()

	proxyForever(proxy.NewProxy(host, container))
}
