// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package socket provides a portable interface for socket system
// calls.
package socket // import "golang.org/x/net/internal/socket"

import "errors"

// An Option represents a sticky socket option.
type Option struct {
	Level int // level
	Name  int // name; must be equal or greater than 1
	Len   int // length of value in bytes; must be equal or greater than 1
}

// Get reads a value for the option from the kernel.
// It returns the number of bytes written into b.
func (o *Option) Get(c *Conn, b []byte) (int, error) {
	if o.Name < 1 || o.Len < 1 {
		return 0, errors.New("invalid option")
	}
	if len(b) < o.Len {
		return 0, errors.New("short buffer")
	}
	return o.get(c, b)
}

// GetInt returns an integer value for the option.
//
// The Len field of Option must be either 1 or 4.
func (o *Option) GetInt(c *Conn) (int, error) {
	if o.Len != 1 && o.Len != 4 {
		return 0, errors.New("invalid option")
	}
	var b []byte
	var bb [4]byte
	if o.Len == 1 {
		b = bb[:1]
	} else {
		b = bb[:4]
	}
	n, err := o.get(c, b)
	if err != nil {
		return 0, err
	}
	if n != o.Len {
		return 0, errors.New("invalid option length")
	}
	if o.Len == 1 {
		return int(b[0]), nil
	}
	return int(NativeEndian.Uint32(b[:4])), nil
}

// Set writes the option and value to the kernel.
func (o *Option) Set(c *Conn, b []byte) error {
	if o.Name < 1 || o.Len < 1 {
		return errors.New("invalid option")
	}
	if len(b) < o.Len {
		return errors.New("short buffer")
	}
	return o.set(c, b)
}

// SetInt writes the option and value to the kernel.
//
// The Len field of Option must be either 1 or 4.
func (o *Option) SetInt(c *Conn, v int) error {
	if o.Len != 1 && o.Len != 4 {
		return errors.New("invalid option")
	}
	var b []byte
	if o.Len == 1 {
		b = []byte{byte(v)}
	} else {
		var bb [4]byte
		NativeEndian.PutUint32(bb[:o.Len], uint32(v))
		b = bb[:4]
	}
	return o.set(c, b)
}
