// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris,!windows

package ipv4

import "golang.org/x/net/internal/socket"

func setControlMessage(c *socket.Conn, opt *rawOpt, cf ControlFlags, on bool) error {
	return errOpNoSupport
}

func newControlMessage(opt *rawOpt) []byte {
	return nil
}

func parseControlMessage(b []byte) (*ControlMessage, error) {
	return nil, errOpNoSupport
}

func marshalControlMessage(cm *ControlMessage) []byte {
	return nil
}
