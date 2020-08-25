go-dhclient
===========

[![CircleCI](https://circleci.com/gh/digineo/go-dhclient/tree/master.svg?style=shield)](https://circleci.com/gh/digineo/go-dhclient/tree/master)
[![Codecov](http://codecov.io/github/digineo/go-dhclient/coverage.svg?branch=master)](http://codecov.io/github/digineo/go-dhclient?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/digineo/go-dhclient)](https://goreportcard.com/report/github.com/digineo/go-dhclient)

`go-dhclient` is a DHCPv4 client library written in Go.
It uses raw sockets and binds them to a specific interface.
Callback functions are triggered on binding or expiration of a lease.

See [main.go](cmd/dhclient/main.go) for example code.

## Dependencies

* [github.com/google/gopacket](https://github.com/google/gopacket) for serializing/unserializing of DHCP packets
* [github.com/mdlayher/raw](https://github.com/mdlayher/raw) for raw sockets
