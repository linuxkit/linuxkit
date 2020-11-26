package funk

import (
	"reflect"
)

func equal(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	return reflect.DeepEqual(expected, actual)

}

func sliceElem(rtype reflect.Type) reflect.Type {
	for {
		if rtype.Kind() != reflect.Slice && rtype.Kind() != reflect.Array {
			return rtype
		}

		rtype = rtype.Elem()
	}
}

func redirectValue(value reflect.Value) reflect.Value {
	for {
		if !value.IsValid() || value.Kind() != reflect.Ptr {
			return value
		}

		res := reflect.Indirect(value)

		// Test for a circular type.
		if res.Kind() == reflect.Ptr && value.Pointer() == res.Pointer() {
			return value
		}

		value = res
	}
}

func makeSlice(value reflect.Value, values ...int) reflect.Value {
	sliceType := sliceElem(value.Type())

	size := value.Len()
	cap := size

	if len(values) > 0 {
		size = values[0]
	}

	if len(values) > 1 {
		cap = values[1]
	}

	return reflect.MakeSlice(reflect.SliceOf(sliceType), size, cap)
}
