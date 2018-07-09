package main

// Log the kernel log buffer (from /dev/kmsg)

import (
	"fmt"
	"log"
	"time"

	"github.com/euank/go-kmsg-parser/kmsgparser"
)

func main() {
	parser, err := kmsgparser.NewParser()
	if err != nil {
		log.Fatalf("unable to create parser: %v", err)
	}
	defer parser.Close()

	kmsg := parser.Parse()

	for msg := range kmsg {
		fmt.Printf("(%d) - %s: %s", msg.SequenceNumber, msg.Timestamp.Format(time.RFC3339Nano), msg.Message)
	}
}
