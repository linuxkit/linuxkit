package main

import (
	"os"
	"path"
)

func main() {
	if path.Base(os.Args[0]) == "proxy-vsockd" {
		manyPorts()
		return
	}
	onePort()
}
