package funk

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	i     interface{}
	zeros = []interface{}{
		false,
		byte(0),
		complex64(0),
		complex128(0),
		float32(0),
		float64(0),
		int(0),
		int8(0),
		int16(0),
		int32(0),
		int64(0),
		rune(0),
		uint(0),
		uint8(0),
		uint16(0),
		uint32(0),
		uint64(0),
		uintptr(0),
		"",
		[0]interface{}{},
		[]interface{}(nil),
		struct{ x int }{},
		(*interface{})(nil),
		(func())(nil),
		nil,
		interface{}(nil),
		map[interface{}]interface{}(nil),
		(chan interface{})(nil),
		(<-chan interface{})(nil),
		(chan<- interface{})(nil),
	}
	nonZeros = []interface{}{
		true,
		byte(1),
		complex64(1),
		complex128(1),
		float32(1),
		float64(1),
		int(1),
		int8(1),
		int16(1),
		int32(1),
		int64(1),
		rune(1),
		uint(1),
		uint8(1),
		uint16(1),
		uint32(1),
		uint64(1),
		uintptr(1),
		"s",
		[1]interface{}{1},
		[]interface{}{},
		struct{ x int }{1},
		(*interface{})(&i),
		(func())(func() {}),
		interface{}(1),
		map[interface{}]interface{}{},
		(chan interface{})(make(chan interface{})),
		(<-chan interface{})(make(chan interface{})),
		(chan<- interface{})(make(chan interface{})),
	}
)

func TestPtrOf(t *testing.T) {
	is := assert.New(t)

	type embedType struct {
		value int
	}

	type anyType struct {
		value    int
		embed    embedType
		embedPtr *embedType
	}

	any := anyType{value: 1}
	anyPtr := &anyType{value: 1}

	results := []interface{}{
		PtrOf(any),
		PtrOf(anyPtr),
	}

	for _, r := range results {
		is.Equal(1, r.(*anyType).value)
		is.Equal(reflect.ValueOf(r).Kind(), reflect.Ptr)
		is.Equal(reflect.ValueOf(r).Type().Elem(), reflect.TypeOf(anyType{}))
	}

	anyWithEmbed := anyType{value: 1, embed: embedType{value: 2}}
	anyWithEmbedPtr := anyType{value: 1, embedPtr: &embedType{value: 2}}

	results = []interface{}{
		PtrOf(anyWithEmbed.embed),
		PtrOf(anyWithEmbedPtr.embedPtr),
	}

	for _, r := range results {
		is.Equal(2, r.(*embedType).value)
		is.Equal(reflect.ValueOf(r).Kind(), reflect.Ptr)
		is.Equal(reflect.ValueOf(r).Type().Elem(), reflect.TypeOf(embedType{}))
	}
}

func TestSliceOf(t *testing.T) {
	is := assert.New(t)

	f := &Foo{
		ID:        1,
		FirstName: "Dark",
		LastName:  "Vador",
		Age:       30,
		Bar: &Bar{
			Name: "Test",
		},
	}

	result := SliceOf(f)

	resultType := reflect.TypeOf(result)

	is.True(resultType.Kind() == reflect.Slice)
	is.True(resultType.Elem().Kind() == reflect.Ptr)

	elemType := resultType.Elem().Elem()

	is.True(elemType.Kind() == reflect.Struct)

	value := reflect.ValueOf(result)

	is.Equal(value.Len(), 1)

	_, ok := value.Index(0).Interface().(*Foo)

	is.True(ok)
}

func TestRandomInt(t *testing.T) {
	is := assert.New(t)

	is.True(RandomInt(0, 10) <= 10)
}

func TestShard(t *testing.T) {
	is := assert.New(t)

	tokey := "e89d66bdfdd4dd26b682cc77e23a86eb"

	is.Equal(Shard(tokey, 1, 2, false), []string{"e", "8", "e89d66bdfdd4dd26b682cc77e23a86eb"})
	is.Equal(Shard(tokey, 2, 2, false), []string{"e8", "9d", "e89d66bdfdd4dd26b682cc77e23a86eb"})
	is.Equal(Shard(tokey, 2, 3, true), []string{"e8", "9d", "66", "bdfdd4dd26b682cc77e23a86eb"})
}

func TestRandomString(t *testing.T) {
	is := assert.New(t)

	is.Len(RandomString(10), 10)

	result := RandomString(10, []rune("abcdefg"))

	is.Len(result, 10)

	for _, char := range result {
		is.True(char >= []rune("a")[0] && char <= []rune("g")[0])
	}
}

func TestIsEmpty(t *testing.T) {
	is := assert.New(t)

	chWithValue := make(chan struct{}, 1)
	chWithValue <- struct{}{}
	var tiP *time.Time
	var tiNP time.Time
	var s *string
	var f *os.File
	ptrs := new(string)
	*ptrs = ""

	is.True(IsEmpty(ptrs), "Nil string pointer is empty")
	is.True(IsEmpty(""), "Empty string is empty")
	is.True(IsEmpty(nil), "Nil is empty")
	is.True(IsEmpty([]string{}), "Empty string array is empty")
	is.True(IsEmpty(0), "Zero int value is empty")
	is.True(IsEmpty(false), "False value is empty")
	is.True(IsEmpty(make(chan struct{})), "Channel without values is empty")
	is.True(IsEmpty(s), "Nil string pointer is empty")
	is.True(IsEmpty(f), "Nil os.File pointer is empty")
	is.True(IsEmpty(tiP), "Nil time.Time pointer is empty")
	is.True(IsEmpty(tiNP), "time.Time is empty")

	is.False(NotEmpty(ptrs), "Nil string pointer is empty")
	is.False(NotEmpty(""), "Empty string is empty")
	is.False(NotEmpty(nil), "Nil is empty")
	is.False(NotEmpty([]string{}), "Empty string array is empty")
	is.False(NotEmpty(0), "Zero int value is empty")
	is.False(NotEmpty(false), "False value is empty")
	is.False(NotEmpty(make(chan struct{})), "Channel without values is empty")
	is.False(NotEmpty(s), "Nil string pointer is empty")
	is.False(NotEmpty(f), "Nil os.File pointer is empty")
	is.False(NotEmpty(tiP), "Nil time.Time pointer is empty")
	is.False(NotEmpty(tiNP), "time.Time is empty")

	is.False(IsEmpty("something"), "Non Empty string is not empty")
	is.False(IsEmpty(errors.New("something")), "Non nil object is not empty")
	is.False(IsEmpty([]string{"something"}), "Non empty string array is not empty")
	is.False(IsEmpty(1), "Non-zero int value is not empty")
	is.False(IsEmpty(true), "True value is not empty")
	is.False(IsEmpty(chWithValue), "Channel with values is not empty")

	is.True(NotEmpty("something"), "Non Empty string is not empty")
	is.True(NotEmpty(errors.New("something")), "Non nil object is not empty")
	is.True(NotEmpty([]string{"something"}), "Non empty string array is not empty")
	is.True(NotEmpty(1), "Non-zero int value is not empty")
	is.True(NotEmpty(true), "True value is not empty")
	is.True(NotEmpty(chWithValue), "Channel with values is not empty")
}

func TestIsZero(t *testing.T) {
	is := assert.New(t)

	for _, test := range zeros {
		is.True(IsZero(test))
	}

	for _, test := range nonZeros {
		is.False(IsZero(test))
	}
}

func TestAny(t *testing.T) {
	is := assert.New(t)

	is.True(Any(true, false))
	is.True(Any(true, true))
	is.False(Any(false, false))
	is.False(Any("", nil, false))
}

func TestAll(t *testing.T) {
	is := assert.New(t)

	is.False(All(true, false))
	is.True(All(true, true))
	is.False(All(false, false))
	is.False(All("", nil, false))
	is.True(All("foo", true, 3))
}

func TestIsIteratee(t *testing.T) {
	is := assert.New(t)

	is.False(IsIteratee(nil))
}
