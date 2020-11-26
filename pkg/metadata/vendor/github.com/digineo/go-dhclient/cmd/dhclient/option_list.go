package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/gopacket/layers"

	dhclient "github.com/digineo/go-dhclient"
)

type optionList []dhclient.Option

func (m *optionList) Set(arg string) error {
	i := strings.Index(arg, ",")
	if i < 0 {
		return errors.New("invalid \"code,value\" pair")
	}

	code, err := strconv.Atoi(arg[:i])
	if err != nil {
		return fmt.Errorf("option code \"%s\" is invalid", arg[:i])
	}

	value := arg[i+1:]
	var data []byte
	if strings.HasPrefix(value, "0x") {
		data, err = hex.DecodeString(value[2:])
		if err != nil {
			return err
		}
	} else {
		data = []byte(value)
	}

	*m = append(*m, dhclient.Option{Type: layers.DHCPOpt(code), Data: data})
	return nil
}

func (m optionList) String() string {
	out := new(bytes.Buffer)
	for _, option := range m {
		if out.Len() != 0 {
			out.WriteByte(' ')
		}
		fmt.Fprintf(out, "%d,0x%x", option.Type, option.Data)
	}
	return out.String()
}
