package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/docker/infrakit/pkg/types"
	"github.com/jmespath/go-jmespath"
)

// DeepCopyObject makes a deep copy of the argument, using encoding/gob encode/decode.
func DeepCopyObject(from interface{}) (interface{}, error) {
	var mod bytes.Buffer
	enc := json.NewEncoder(&mod)
	dec := json.NewDecoder(&mod)
	err := enc.Encode(from)
	if err != nil {
		return nil, err
	}

	copy := reflect.New(reflect.TypeOf(from))
	err = dec.Decode(copy.Interface())
	if err != nil {
		return nil, err
	}
	return reflect.Indirect(copy).Interface(), nil
}

// QueryObject applies a JMESPath query specified by the expression, against the target object.
func QueryObject(exp string, target interface{}) (interface{}, error) {
	query, err := jmespath.Compile(exp)
	if err != nil {
		return nil, err
	}
	return query.Search(target)
}

// SplitLines splits the input into a string slice.
func SplitLines(o interface{}) ([]string, error) {
	ret := []string{}
	switch o := o.(type) {
	case string:
		return strings.Split(o, "\n"), nil
	case []byte:
		return strings.Split(string(o), "\n"), nil
	}
	return ret, fmt.Errorf("not-supported-value-type")
}

// FromJSON decode the input JSON encoded as string or byte slice into a map.
func FromJSON(o interface{}) (interface{}, error) {
	var ret interface{}
	switch o := o.(type) {
	case string:
		err := json.Unmarshal([]byte(o), &ret)
		return ret, err
	case []byte:
		err := json.Unmarshal(o, &ret)
		return ret, err
	case *types.Any:
		err := json.Unmarshal(o.Bytes(), &ret)
		return ret, err
	}
	return ret, fmt.Errorf("not-supported-value-type")
}

// ToJSON encodes the input struct into a JSON string.
func ToJSON(o interface{}) (string, error) {
	buff, err := json.MarshalIndent(o, "", "  ")
	return string(buff), err
}

// ToJSONFormat encodes the input struct into a JSON string with format prefix, and indent.
func ToJSONFormat(prefix, indent string, o interface{}) (string, error) {
	buff, err := json.MarshalIndent(o, prefix, indent)
	return string(buff), err
}

// FromMap decodes map into raw struct
func FromMap(m map[string]interface{}, raw interface{}) error {
	// The safest way, but the slowest, is to just marshal and unmarshal back
	buff, err := ToJSON(m)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(buff), raw)
}

// ToMap encodes the input as a map
func ToMap(raw interface{}) (map[string]interface{}, error) {
	buff, err := ToJSON(raw)
	if err != nil {
		return nil, err
	}
	out, err := FromJSON(buff)
	return out.(map[string]interface{}), err
}

// UnixTime returns a timestamp in unix time
func UnixTime() interface{} {
	return time.Now().Unix()
}

// IndexOf returns the index of search in array.  -1 if not found or array is not iterable.  An optional true will
// turn on strict type check while by default string representations are used to compare values.
func IndexOf(srch interface{}, array interface{}, strictOptional ...bool) int {
	strict := false
	if len(strictOptional) > 0 {
		strict = strictOptional[0]
	}
	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)
		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(srch, s.Index(i).Interface()) {
				return i
			}
			if !strict {
				// by string value which is useful for text based compares
				search := reflect.Indirect(reflect.ValueOf(srch)).Interface()
				value := reflect.Indirect(s.Index(i)).Interface()
				searchStr := fmt.Sprintf("%v", search)
				check := fmt.Sprintf("%v", value)
				if searchStr == check {
					return i
				}
			}
		}
	}
	return -1
}

// DefaultFuncs returns a list of default functions for binding in the template
func (t *Template) DefaultFuncs() []Function {
	return []Function{
		{
			Name: "source",
			Description: []string{
				"Source / evaluate the template at the input location (as URL).",
				"This will make all of the global variables declared there visible in this template's context.",
				"Similar to 'source' in bash, sourcing another template means applying it in the same context ",
				"as the calling template.  The context (e.g. variables) of the calling template as a result can be mutated.",
			},
			Func: func(p string, opt ...interface{}) (string, error) {
				var o interface{}
				if len(opt) > 0 {
					o = opt[0]
				}
				loc := p
				if strings.Index(loc, "str://") == -1 {
					buff, err := getURL(t.url, p)
					if err != nil {
						return "", err
					}
					loc = buff
				}
				sourced, err := NewTemplate(loc, t.options)
				if err != nil {
					return "", err
				}
				// set this as the parent of the sourced template so its global can mutate the globals in this
				sourced.parent = t
				sourced.forkFrom(t)
				sourced.context = t.context

				if o == nil {
					o = sourced.context
				}
				// TODO(chungers) -- let the sourced template define new functions that can be called in the parent.
				return sourced.Render(o)
			},
		},
		{
			Name: "include",
			Description: []string{
				"Render content found at URL as template and include here.",
				"The optional second parameter is the context to use when rendering the template.",
				"Conceptually similar to exec in bash, where the template included is applied using a fork ",
				"of current context in the calling template.  Any mutations to the context via 'global' will not ",
				"be visible in the calling template's context.",
			},
			Func: func(p string, opt ...interface{}) (string, error) {
				var o interface{}
				if len(opt) > 0 {
					o = opt[0]
				}
				loc := p
				if strings.Index(loc, "str://") == -1 {
					buff, err := getURL(t.url, p)
					if err != nil {
						return "", err
					}
					loc = buff
				}
				included, err := NewTemplate(loc, t.options)
				if err != nil {
					return "", err
				}
				dotCopy, err := included.forkFrom(t)
				if err != nil {
					return "", err
				}
				included.context = dotCopy

				if o == nil {
					o = included.context
				}

				return included.Render(o)
			},
		},
		{
			Name: "loop",
			Description: []string{
				"Loop generates a slice of length specified by the input. For use like {{ range loop 5 }}...{{ end }}",
			},
			Func: func(c int) []struct{} {
				return make([]struct{}, c)
			},
		},
		{
			Name: "global",
			Description: []string{
				"Sets a global variable named after the first argument, with the value as the second argument.",
				"This is similar to def (which sets the default value).",
				"Global variables are propagated to all templates that are rendered via the 'include' function.",
			},
			Func: func(n string, v interface{}) Void {
				t.Global(n, v)
				return voidValue
			},
		},
		{
			Name: "def",
			Description: []string{
				"Defines a variable with the first argument as name and last argument value as the default.",
				"It's also ok to pass a third optional parameter, in the middle, as the documentation string.",
			},
			Func: func(name string, args ...interface{}) (Void, error) {
				if _, has := t.defaults[name]; has {
					// not sure if this is good, but should complain loudly
					return voidValue, fmt.Errorf("already defined: %v", name)
				}
				var doc string
				var value interface{}
				switch len(args) {
				case 1:
					// just value, no docs
					value = args[0]
				case 2:
					// docs and value
					doc = fmt.Sprintf("%v", args[0])
					value = args[1]
				}
				t.Def(name, value, doc)
				return voidValue, nil
			},
		},
		{
			Name: "ref",
			Description: []string{
				"References / gets the variable named after the first argument.",
				"The values must be set first by either def or global.",
			},
			Func: t.Ref,
		},
		{
			Name: "q",
			Description: []string{
				"Runs a JMESPath (http://jmespath.org/) query (first arg) on the object (second arg).",
				"The return value is an object which needs to be rendered properly for the format of the document.",
				"Example: {{ include \"https://httpbin.org/get\" | from_json | q \"origin\" }} returns the origin of http request.",
			},
			Func: QueryObject,
		},
		{
			Name: "to_json",
			Description: []string{
				"Encodes the input as a JSON string",
				"This is useful for taking an object (interface{}) and render it inline as proper JSON.",
				"Example: {{ include \"https://httpbin.org/get\" | from_json | to_json }}",
			},
			Func: ToJSON,
		},
		{
			Name: "jsonEncode",
			Description: []string{
				"Encodes the input as a JSON string",
				"This is useful for taking an object (interface{}) and render it inline as proper JSON.",
				"Example: {{ include \"https://httpbin.org/get\" | from_json | to_json }}",
			},
			Func: ToJSON,
		},
		{
			Name: "to_json_format",
			Description: []string{
				"Encodes the input as a JSON string with first arg as prefix, second arg the indentation, then the object",
			},
			Func: ToJSONFormat,
		},
		{
			Name: "jsonEncodeIndent",
			Description: []string{
				"Encodes the input as a JSON string with first arg as prefix, second arg the indentation, then the object",
			},
			Func: ToJSONFormat,
		},
		{
			Name: "from_json",
			Description: []string{
				"Decodes the input (first arg) into a structure (a map[string]interface{} or []interface{}).",
				"This is useful for parsing arbitrary resources in JSON format as object.  The object is the queryable via 'q'",
				"For example: {{ include \"https://httpbin.org/get\" | from_json | q \"origin\" }} returns the origin of request.",
			},
			Func: FromJSON,
		},
		{
			Name: "jsonDecode",
			Description: []string{
				"Decodes the input (first arg) into a structure (a map[string]interface{} or []interface{}).",
				"This is useful for parsing arbitrary resources in JSON format as object.  The object is the queryable via 'q'",
				"For example: {{ include \"https://httpbin.org/get\" | from_json | q \"origin\" }} returns the origin of request.",
			},
			Func: FromJSON,
		},
		{
			Name: "unixtime",
			Description: []string{
				"Returns the unix timestamp as the number of seconds elapsed since January 1, 1970 UTC.",
			},
			Func: UnixTime,
		},
		{
			Name: "lines",
			Description: []string{
				"Splits the input string (first arg) into a slice by '\n'",
			},
			Func: SplitLines,
		},
		{
			Name: "index_of",
			Description: []string{
				"Returns the index of first argument in the second argument which is a slice.",
				"Example: {{ index_of \"foo\" (from_json \"[\"bar\",\"foo\",\"baz\"]\") }} returns 1 (int).",
			},
			Func: IndexOf,
		},
		{
			Name: "indexOf",
			Description: []string{
				"Returns the index of first argument in the second argument which is a slice.",
				"Example: {{ index_of \"foo\" (from_json \"[\"bar\",\"foo\",\"baz\"]\") }} returns 1 (int).",
			},
			Func: IndexOf,
		},
	}
}
