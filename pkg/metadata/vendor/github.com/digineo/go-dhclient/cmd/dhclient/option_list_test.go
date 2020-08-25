package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionList(t *testing.T) {
	assert := assert.New(t)

	m := optionList{}

	assert.EqualError(m.Set("foo"), "invalid \"code,value\" pair")
	assert.EqualError(m.Set(","), "option code \"\" is invalid")
	assert.EqualError(m.Set("0x12,foo"), "option code \"0x12\" is invalid")

	assert.NoError(m.Set("7,0x01020304"))
	assert.NoError(m.Set("65,foo"))
	assert.Len(m, 2)
	assert.Equal("7,0x01020304 65,0x666f6f", m.String())
}
