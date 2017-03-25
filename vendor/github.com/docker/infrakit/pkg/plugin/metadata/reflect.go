package metadata

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/infrakit/pkg/types"
)

var (
	indexRoot     = "\\[(([+|-]*[0-9]+)|((.*)=(.*)))\\]$"
	arrayIndexExp = regexp.MustCompile("(.*)" + indexRoot)
	indexExp      = regexp.MustCompile("^" + indexRoot)
)

// Put sets the attribute of an object at path to the given value
func Put(path []string, value interface{}, object map[string]interface{}) bool {
	return put(path, value, object)
}

// Get returns the attribute of the object at path
func Get(path []string, object interface{}) interface{} {
	return get(path, object)
}

// GetValue returns the attribute of the object at path, as serialized blob
func GetValue(path []string, object interface{}) (*types.Any, error) {
	if any, is := object.(*types.Any); is {
		return any, nil
	}
	return types.AnyValue(Get(path, object))
}

// List lists the members at the path
func List(path []string, object interface{}) []string {
	list := []string{}
	v := get(path, object)
	if v == nil {
		return list
	}

	val := reflect.Indirect(reflect.ValueOf(v))

	if any, is := v.(*types.Any); is {
		var temp interface{}
		if err := any.Decode(&temp); err == nil {
			val = reflect.ValueOf(temp)
		}
	}

	switch val.Kind() {
	case reflect.Slice:
		// this is a slice, so return the name as '[%d]'
		for i := 0; i < val.Len(); i++ {
			list = append(list, fmt.Sprintf("[%d]", i))
		}

	case reflect.Map:
		for _, k := range val.MapKeys() {
			list = append(list, k.String())
		}

	case reflect.Struct:
		vt := val.Type()
		for i := 0; i < vt.NumField(); i++ {
			if vt.Field(i).PkgPath == "" {
				list = append(list, vt.Field(i).Name)
			}
		}
	}

	sort.Strings(list)
	return list
}

func put(p []string, value interface{}, store map[string]interface{}) bool {
	if len(p) == 0 {
		return false
	}

	key := p[0]
	if key == "" {
		return put(p[1:], value, store)
	}
	// check if key is an array index of the form <1>[<2>]
	matches := arrayIndexExp.FindStringSubmatch(key)
	if len(matches) > 2 && matches[1] != "" {
		key = matches[1]
		p = append([]string{key, fmt.Sprintf("[%s]", matches[2])}, p[1:]...)
		return put(p, value, store)
	}

	s := reflect.Indirect(reflect.ValueOf(store))
	switch s.Kind() {
	case reflect.Slice:
		return false // not supported

	case reflect.Map:
		if reflect.TypeOf(p[0]).AssignableTo(s.Type().Key()) {
			m := s.MapIndex(reflect.ValueOf(p[0]))
			if !m.IsValid() {
				m = reflect.ValueOf(map[string]interface{}{})
				s.SetMapIndex(reflect.ValueOf(p[0]), m)
			}
			if len(p) > 1 {
				return put(p[1:], value, m.Interface().(map[string]interface{}))
			}
			s.SetMapIndex(reflect.ValueOf(p[0]), reflect.ValueOf(value))
			return true
		}
	}
	return false
}

func get(path []string, object interface{}) (value interface{}) {
	if f, is := object.(func() interface{}); is {
		object = f()
	}

	if len(path) == 0 {
		return object
	}

	if any, is := object.(*types.Any); is {
		var temp interface{}
		if err := any.Decode(&temp); err == nil {
			return get(path, temp)
		}
		return nil
	}

	key := path[0]

	switch key {
	case ".":
		return object
	case "":
		return get(path[1:], object)
	}

	// check if key is an array index of the form <1>[<2>]
	matches := arrayIndexExp.FindStringSubmatch(key)
	if len(matches) > 2 && matches[1] != "" {
		key = matches[1]
		path = append([]string{key, fmt.Sprintf("[%s]", matches[2])}, path[1:]...)
		return get(path, object)
	}

	v := reflect.Indirect(reflect.ValueOf(object))
	switch v.Kind() {
	case reflect.Slice:
		i := 0
		matches = indexExp.FindStringSubmatch(key)
		if len(matches) > 0 {
			if matches[2] != "" {
				// numeric index
				if index, err := strconv.Atoi(matches[1]); err == nil {
					switch {
					case index >= 0 && v.Len() > index:
						i = index
					case index < 0 && v.Len() > -index: // negative index like python
						i = v.Len() + index
					}
				}
				return get(path[1:], v.Index(i).Interface())

			} else if matches[3] != "" {
				// equality search index for 'field=check'
				lhs := matches[4] // supports another select expression for extracting deeply from the struct
				rhs := matches[5]
				// loop through the array looking for field that matches the check value
				for j := 0; j < v.Len(); j++ {
					if el := get(tokenize(lhs), v.Index(j).Interface()); el != nil {
						if fmt.Sprintf("%v", el) == rhs {
							return get(path[1:], v.Index(j).Interface())
						}
					}
				}
			}
		}
	case reflect.Map:
		value := v.MapIndex(reflect.ValueOf(key))
		if value.IsValid() {
			return get(path[1:], value.Interface())
		}
	case reflect.Struct:
		fv := v.FieldByName(key)
		if !fv.IsValid() {
			return nil
		}
		if !fv.CanInterface() {
			return nil
		}
		return get(path[1:], fv.Interface())
	}
	return nil
}

// With quoting to support azure rm type names: e.g. Microsoft.Network/virtualNetworks
// This will split a sting like /Resources/'Microsoft.Network/virtualNetworks'/managerSubnet/Name" into
// [ , Resources, Microsoft.Network/virtualNetworks, managerSubnet, Name]
func tokenize(s string) []string {
	if len(s) == 0 {
		return []string{}
	}

	a := []string{}
	start := 0
	quoted := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '/':
			if !quoted {
				a = append(a, strings.Replace(s[start:i], "'", "", -1))
				start = i + 1
			}
		case '\'':
			quoted = !quoted
		}
	}
	if start < len(s)-1 {
		a = append(a, strings.Replace(s[start:], "'", "", -1))
	}

	return a
}
