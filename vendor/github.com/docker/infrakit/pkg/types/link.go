package types

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

// Link is a struct that represents an association between an infrakit managed resource
// and an entity in some other system.  The mechanism of linkage is via labels or tags
// on both sides.
type Link struct {
	value   string
	context string
}

// NewLink creates a link
func NewLink() *Link {
	return &Link{
		value: randomAlphaNumericString(16),
	}
}

// NewLinkFromMap constructs a link from data in the map
func NewLinkFromMap(m map[string]string) *Link {
	l := &Link{}
	if v, has := m["infrakit-link"]; has {
		l.value = v
	}

	if v, has := m["infrakit-link-context"]; has {
		l.context = v
	}
	return l
}

// Valid returns true if the link value is set
func (l Link) Valid() bool {
	return l.value != ""
}

// Value returns the value of the link
func (l Link) Value() string {
	return l.value
}

// Label returns the label to look for the link
func (l Link) Label() string {
	return "infrakit-link"
}

// Context returns the context of the link
func (l Link) Context() string {
	return l.context
}

// WithContext sets a context for this link
func (l *Link) WithContext(s string) *Link {
	l.context = s
	return l
}

// KVPairs returns the link representation as a slice of Key=Value pairs
func (l *Link) KVPairs() []string {
	out := []string{}
	for k, v := range l.Map() {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}

// Map returns a representation that is easily converted to JSON or YAML
func (l *Link) Map() map[string]string {
	return map[string]string{
		"infrakit-link":         l.value,
		"infrakit-link-context": l.context,
	}
}

// WriteMap writes to the target map.  This will overwrite values of same key
func (l *Link) WriteMap(target map[string]string) {
	for k, v := range l.Map() {
		target[k] = v
	}
}

// InMap returns true if the link is contained in the map
func (l *Link) InMap(m map[string]string) bool {
	c, has := m["infrakit-link-context"]
	if !has {
		return false
	}
	if c != l.context {
		return false
	}

	v, has := m["infrakit-link"]
	if !has {
		return false
	}
	return v == l.value
}

// Equal returns true if the links are the same - same value and context
func (l *Link) Equal(other Link) bool {
	return l.value == other.value && l.context == other.context
}

// randomAlphaNumericString generates a non-secure random alpha-numeric string of a given length.
func randomAlphaNumericString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
