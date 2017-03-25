package template

import (
	"fmt"
	"reflect"
	"strings"
)

// UpdateDocumentation uses reflection to generate documentation on usage and function signature.
func UpdateDocumentation(in []Function) []Function {
	out := []Function{}
	for _, f := range in {
		copy := f
		copy.Function = functionSignature(f.Name, f.Func)
		copy.Usage = functionUsage(f.Name, f.Func)
		if len(f.Description) == 0 {
			copy.Description = []string{"None"}
		}
		out = append(out, copy)
	}
	return out
}

func isFunc(f interface{}) (string, bool) {
	if f == nil {
		return "no-function", false
	}

	ft := reflect.TypeOf(f)
	if ft.Kind() != reflect.Func {
		return "not-a-function", false
	}
	return ft.String(), true
}

func functionSignature(name string, f interface{}) string {
	s, is := isFunc(f)
	if !is {
		return s
	}
	return s
}

func functionUsage(name string, f interface{}) string {
	if s, is := isFunc(f); !is {
		return s
	}

	ft := reflect.TypeOf(f)
	if ft.Kind() != reflect.Func {
		return "not-a-function"
	}

	args := make([]string, ft.NumIn())
	for i := 0; i < len(args); i++ {
		t := ft.In(i)

		v := ""
		switch {
		case t == reflect.TypeOf(""):
			v = fmt.Sprintf("\"%s\"", t.Name())

		case t.Kind() == reflect.Slice && i == len(args)-1:
			tt := t.Elem().Name()
			if t.Elem() == reflect.TypeOf("") {
				tt = fmt.Sprintf("\"%s\"", t.Name())
			}
			v = fmt.Sprintf("[ %s ... ]", tt)
		case t.String() == "interface {}":
			v = "any"
		default:
			v = strings.Replace(t.String(), " ", "", -1)
		}

		args[i] = v
	}

	arglist := strings.Join(args, " ")
	if len(arglist) > 0 {
		arglist = arglist + " "
	}
	return fmt.Sprintf("{{ %s %s}}", name, arglist)
}
