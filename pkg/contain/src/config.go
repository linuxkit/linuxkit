package main

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type ContainConfig struct {
	Namespace          string    `yaml:"namespace"`
	HostMountContainer []Contain `yaml:"configs"`
	Socket             string    `yaml:"socket"`
}
type Contain struct {
	Command     []string `yaml:"command"`
	Name        string   `yaml:"name"`
	Image       string   `yaml:"image"`
	Source      string   `yaml:"source"`
	Destination string   `yaml:"destination"`
}

func loadConfig(path string) (*ContainConfig, error) {
	var config = &ContainConfig{}
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		return config, err
	}
	return config, err
}

func getContain(args []string, containConfig *ContainConfig) (error, Contain) {
	for _, config := range containConfig.HostMountContainer {
		if checkEqual(config.Command, args) {
			return nil, config
		}
	}
	return errors.New(fmt.Sprintln("command not found:", args)), Contain{}
}

func checkEqual(param1 []string, param2 []string) bool {
	for i, _ := range param1 {
		if param1[i] != param2[i] {
			return false
		}
	}
	return true
}
