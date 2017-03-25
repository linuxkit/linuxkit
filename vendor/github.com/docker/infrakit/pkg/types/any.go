package types

import (
	"encoding/json"
)

// Any is the raw configuration for the plugin
type Any json.RawMessage

// AnyString returns an Any from a string that represents the marshaled/encoded data
func AnyString(s string) *Any {
	return AnyBytes([]byte(s))
}

// AnyBytes returns an Any from the encoded message bytes
func AnyBytes(data []byte) *Any {
	any := &Any{}
	*any = data
	return any
}

// AnyCopy makes a copy of the data in the given ptr.
func AnyCopy(any *Any) *Any {
	if any == nil {
		return &Any{}
	}
	return AnyBytes(any.Bytes())
}

// AnyValue returns an Any from a value by marshaling / encoding the input
func AnyValue(v interface{}) (*Any, error) {
	if v == nil {
		return nil, nil // So that any omitempty will see an empty/zero value
	}
	any := &Any{}
	err := any.marshal(v)
	return any, err
}

// AnyValueMust returns an Any from a value by marshaling / encoding the input. It panics if there's error.
func AnyValueMust(v interface{}) *Any {
	any, err := AnyValue(v)
	if err != nil {
		panic(err)
	}
	return any
}

// Decode decodes the any into the input typed struct
func (c *Any) Decode(typed interface{}) error {
	if c == nil || len([]byte(*c)) == 0 {
		return nil // no effect on typed
	}
	return json.Unmarshal([]byte(*c), typed)
}

// marshal populates this raw message with a decoded form of the input struct.
func (c *Any) marshal(typed interface{}) error {
	buff, err := json.MarshalIndent(typed, "", "")
	if err != nil {
		return err
	}
	*c = Any(json.RawMessage(buff))
	return nil
}

// Bytes returns the encoded bytes
func (c *Any) Bytes() []byte {
	if c == nil {
		return nil
	}
	return []byte(*c)
}

// String returns the string representation.
func (c *Any) String() string {
	return string([]byte(*c))
}

// MarshalJSON implements the json Marshaler interface
func (c *Any) MarshalJSON() ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return []byte(*c), nil
}

// UnmarshalJSON implements the json Unmarshaler interface
func (c *Any) UnmarshalJSON(data []byte) error {
	*c = Any(json.RawMessage(data))
	return nil
}
