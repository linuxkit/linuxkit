package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/template"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type rootsFlag []string

func (flag *rootsFlag) String() string {

	if len(*flag) == 0 {
		return ""
	}
	return fmt.Sprint(*flag)
}

func (flag *rootsFlag) Set(value string) error {
	(*flag) = append(*flag, value)
	return nil
}

var roots rootsFlag

func bake(args []string) {
	flags := flag.NewFlagSet("bake", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "USAGE: %s bake [options] <file>[.yml]\n", invoked)
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}

	flags.Var(&roots, "pkgroot", "path to pkg source of moby config")
	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()

	if len(remArgs) == 0 {
		fmt.Println("Please specify atleast one moby configuration file")
		flags.Usage()
		os.Exit(1)
	}

	var repos []string
	for _, repo := range roots {
		info, err := os.Stat(repo)
		if err != nil {
			log.Warnf("Pkgroot parameter \"%v\" is not valid: %v", repo, err)
			continue
		}
		if !info.IsDir() {
			log.Warnf("Pkgroot parameter \"%v\" is not a dir", repo)
			continue
		}
		repos = append(repos, repo)
	}

	for _, r := range Config.Repos {
		info, err := os.Stat(r.Path)
		if err != nil || !info.IsDir() {
			log.Warnln("Pkg repo from global is not valid: ", r.Path)
			continue
		}
		repos = append(repos, r.Path)
	}

	m := GetMoby(remArgs)

	t := template.Template{Moby: m, Repos: repos}
	result, err := t.Bake()
	if err != nil {
		log.Fatalf(err.Error())
	}

	delimeter := fmt.Sprintln("#-----------------------------------------------------------------------------------")

	var comments string
	comments += fmt.Sprintf("###-------------------------------AUTO-GENERATED-------------------------------###\n")
	comments += fmt.Sprintf("#     time: %v\n", time.Now())
	comments += delimeter
	for _, l := range result.Subs {
		comments += fmt.Sprintln("# template: " + l.Template)
		comments += fmt.Sprintln("#   source: " + l.Source)
		comments += fmt.Sprintln("#   result: " + l.Result)
		comments += delimeter
	}

	bytes := []byte(comments)

	yml, err := yaml.Marshal(result.Moby)
	bytes = append(bytes, yml...)

	if err != nil {
		log.Fatalf(err.Error())
		return
	}

	if err != nil {
		log.Fatalf("Cannot open output file: %v", err)
	}

	_, err = os.Stdout.Write(bytes)
	if err != nil {
		log.Fatalf(err.Error())
		return
	}
}
