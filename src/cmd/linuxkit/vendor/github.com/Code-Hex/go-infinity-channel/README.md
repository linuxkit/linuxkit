# go-infinity-channel

[![.github/workflows/test.yml](https://github.com/Code-Hex/go-infinity-channel/actions/workflows/test.yml/badge.svg)](https://github.com/Code-Hex/go-infinity-channel/actions/workflows/test.yml) [![codecov](https://codecov.io/gh/Code-Hex/go-infinity-channel/branch/main/graph/badge.svg?token=Wm7UEwgiZu)](https://codecov.io/gh/Code-Hex/go-infinity-channel) [![Go Reference](https://pkg.go.dev/badge/github.com/Code-Hex/go-infinity-channel.svg)](https://pkg.go.dev/github.com/Code-Hex/go-infinity-channel)

`go-infinity-channel` is a Go package that provides an infinitely queueing channel. This package offers a `Channel` struct, which uses generics to hold data of any type in a queue between an input channel and an output channel. The buffer between the input and output channels queues the incoming data without any limit, making it available for the output channel. This package is useful when there's no need for buffer size constraints or when avoiding blocking during data transfers.

## Synopsis

Try this on [go playground](https://go.dev/play/p/-3ZLmziBYW8)!

```go
package main

import (
	"fmt"
	"time"

	infinity "github.com/Code-Hex/go-infinity-channel"
)

func main() {
	ch := infinity.NewChannel[int]()

	go func() {
		for i := 0; i < 10; i++ {
			ch.In() <- i
			time.Sleep(100 * time.Millisecond)
		}
		ch.Close()
	}()

	for i := range ch.Out() {
		fmt.Println("Received:", i)
	}
}
```

## Requirements

Go 1.18 or later.

## Install

    $ go get github.com/Code-Hex/go-infinity-channel

## Caution

When reading values from the output channel, it is important to use the `for range` loop to avoid panics caused by receiving from a closed channel. Do not use an incremental `for` loop to read a fixed number of items from the output channel, as it may lead to panics if you try to read more items than were sent to the input channel.

**Recommended:**

```go
for item := range ch.Out() {
	fmt.Println(item)
}
```

**Not recommended:**

```go
// Assuming `count` is the number of items you expect to read from the channel
for i := 0; i < count; i++ {
	fmt.Println(<-ch.Out())
}
```

## Author

- [codehex](https://twitter.com/codehex/)
- [ChatGPT](https://chat.openai.com/)
