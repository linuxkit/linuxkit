package main

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpmutil"
)

const device = "/dev/tpm0"

func initTPM() (io.ReadWriteCloser, error) {
	return tpmutil.OpenTPM(device)
}

func tpmRand(tpm io.ReadWriteCloser) (uint64, error) {
	data, err := tpm2.GetRandom(tpm, 8)
	if err != nil {
		return 0, err
	}
	ui, len := binary.Uvarint(data)
	if len <= 0 {
		return 0, errors.New("bad data")
	}
	return ui, nil
}
