package funk

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedirectValue(t *testing.T) {
	is := assert.New(t)

	val := 1

	is.Equal(redirectValue(reflect.ValueOf(&val)).Interface(), 1)
}
