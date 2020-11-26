// Copyright 2019 The GoPacket Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// +build ignore

// This file generates LayersDecoder function for DecodingLayerContainer
// go run gen.go | gofmt > layers_decoder.go
package main

import (
	"fmt"
	"os"
	"time"
)

const headerFmt = `// Copyright 2019 The GoPacket Authors. All rights reserved.

package gopacket

// Created by gen.go, don't edit manually
// Generated at %s

// LayersDecoder returns DecodingLayerFunc for specified
// DecodingLayerContainer, LayerType value to start decoding with and
// some DecodeFeedback.
func LayersDecoder(dl DecodingLayerContainer, first LayerType, df DecodeFeedback) DecodingLayerFunc {
  firstDec, ok := dl.Decoder(first)
  if !ok {
    return func([]byte, *[]LayerType) (LayerType, error) {
      return first, nil
    }
  }
`

var funcBody = `return func(data []byte, decoded *[]LayerType) (LayerType, error) {
  *decoded = (*decoded)[:0] // Truncated decoded layers.
  typ := first
  decoder := firstDec
  for {
    if err := decoder.DecodeFromBytes(data, df); err != nil {
      return LayerTypeZero, err
    }
    *decoded = append(*decoded, typ)
    typ = decoder.NextLayerType()
    if data = decoder.LayerPayload(); len(data) == 0 {
      break
    }
    if decoder, ok = dlc.Decoder(typ); !ok {
      return typ, nil
    }
  }
  return LayerTypeZero, nil
}`

func main() {
	fmt.Fprintf(os.Stderr, "Writing results to stdout\n")
	types := []string{
		"DecodingLayerSparse",
		"DecodingLayerArray",
		"DecodingLayerMap",
	}

	fmt.Printf(headerFmt, time.Now())
	for _, t := range types {
		fmt.Printf("if dlc, ok := dl.(%s); ok {", t)
		fmt.Println(funcBody)
		fmt.Println("}")
	}
	fmt.Println("dlc := dl")
	fmt.Println(funcBody)
	fmt.Println("}")
}
