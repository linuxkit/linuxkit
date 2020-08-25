package main

import (
	"bytes"
	"fmt"
	"strconv"
)

type byteList []byte

func (list *byteList) Set(value string) error {
	u, err := strconv.ParseUint(value, 0, 8)
	if err != nil {
		return err
	}

	*list = append(*list, byte(u))
	return nil
}

func (list *byteList) String() string {
	out := new(bytes.Buffer)
	for _, b := range *list {
		if out.Len() != 0 {
			out.WriteByte(' ')
		}
		fmt.Fprintf(out, "%d", b)
	}
	return out.String()
}
