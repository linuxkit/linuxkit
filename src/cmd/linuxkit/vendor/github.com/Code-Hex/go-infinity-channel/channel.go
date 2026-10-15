// Package infinity provides an unbounded buffered channel implementation.
package infinity

import "sync"

// Channel represents an unbounded buffered channel for values of type T.
type Channel[T any] struct {
	input, output chan T
	length        chan int
	buffer        []T
	once          sync.Once
}

// NewChannel creates a new unbounded buffered channel for values of type T.
func NewChannel[T any]() *Channel[T] {
	ch := &Channel[T]{
		input:  make(chan T),
		output: make(chan T),
		length: make(chan int),
		buffer: []T{},
	}
	go ch.infiniteBuffer()
	return ch
}

// In returns a send-only channel for writing values to the Channel.
func (ch *Channel[T]) In() chan<- T {
	return ch.input
}

// Out returns a receive-only channel for reading values from the Channel.
func (ch *Channel[T]) Out() <-chan T {
	return ch.output
}

// Len returns the current number of elements in the Channel buffer.
func (ch *Channel[T]) Len() int {
	return <-ch.length
}

// Close safely closes the input channel, ensuring that it is closed only once.
// It uses the sync.Once field in the Channel struct to guarantee a single execution.
func (ch *Channel[T]) Close() {
	ch.once.Do(func() {
		close(ch.input)
	})
}

// queue appends a value to the end of the Channel buffer.
func (ch *Channel[T]) queue(v T) { ch.buffer = append(ch.buffer, v) }

// dequeue removes the first value from the Channel buffer.
func (ch *Channel[T]) dequeue() { ch.buffer = ch.buffer[1:] }

// peek returns the first value in the Channel buffer without removing it.
func (ch *Channel[T]) peek() T {
	return ch.buffer[0]
}

// infiniteBuffer is the internal buffer management goroutine for the Channel.
func (ch *Channel[T]) infiniteBuffer() {
	var input, output chan T
	var next T
	input = ch.input

	var zero T

	for input != nil || output != nil {
		select {
		case elem, open := <-input:
			if open {
				ch.queue(elem)
			} else {
				input = nil
			}
		case output <- next:
			ch.dequeue()
		case ch.length <- len(ch.buffer):
		}

		if len(ch.buffer) > 0 {
			output = ch.output
			next = ch.peek()
		} else {
			output = nil
			next = zero
		}
	}

	close(ch.output)
	close(ch.length)
}
