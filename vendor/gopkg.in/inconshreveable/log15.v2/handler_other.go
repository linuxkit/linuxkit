// +build !appengine

package log15

import (
	"sync/atomic"
	"unsafe"
)

// swapHandler wraps another handler that may be swapped out
// dynamically at runtime in a thread-safe fashion.
type swapHandler struct {
	handler unsafe.Pointer
}

func (h *swapHandler) Log(r *Record) error {
	return (*(*Handler)(atomic.LoadPointer(&h.handler))).Log(r)
}

func (h *swapHandler) Swap(newHandler Handler) {
	atomic.StorePointer(&h.handler, unsafe.Pointer(&newHandler))
}
