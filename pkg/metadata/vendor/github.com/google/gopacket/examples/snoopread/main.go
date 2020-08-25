// Copyright 2019 The GoPacket Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// snoopread is a example for read a snoop file using
// gopacket and its subpackages and output the decoded data with a package count
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcapgo"
)

func main() {
	//download snoop from https://wiki.wireshark.org/SampleCaptures
	f, err := os.Open("example.snoop")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer f.Close()
	handle, err := pcapgo.NewSnoopReader(f)
	if err != nil {
		log.Fatal(err)
		return
	}

	lt, err := handle.LinkType()
	if err != nil {
		log.Fatal(err)
		return
	}
	packetSource := gopacket.NewPacketSource(handle, lt)

	cnt := 0
	for packet := range packetSource.Packets() {
		fmt.Println(packet)
		cnt++
	}
	fmt.Printf("Packet count: %d\n", cnt)
}
