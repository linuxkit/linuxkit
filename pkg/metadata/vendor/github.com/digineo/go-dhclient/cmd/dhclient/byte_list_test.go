package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteList(t *testing.T) {
	assert := assert.New(t)

	m := byteList{}

	assert.EqualError(m.Set("foo"), "strconv.ParseUint: parsing \"foo\": invalid syntax")

	assert.NoError(m.Set("65"))
	assert.NoError(m.Set("0x0f"))
	assert.Len(m, 2)
	assert.Equal("65 15", m.String())
}
