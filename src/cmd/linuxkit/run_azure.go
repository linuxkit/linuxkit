package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const ()

// Process the run arguments and execute run
func runAzure(args []string) {
	flags := flag.NewFlagSet("azure", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run azure [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies either the name of an already uploaded\n")
		fmt.Printf("GCP image or the full path to a image file which will be\n")
		fmt.Printf("uploaded before it is run.\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	vhdUri := flags.Parse()

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
}
