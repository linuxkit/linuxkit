package server

import (
	"fmt"
	"net/http"
	"reflect"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/docker/infrakit/pkg/plugin"
	"github.com/docker/infrakit/pkg/rpc"
	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/types"
)

var (
	// Precompute the reflect.Type of error and http.Request -- from gorilla/rpc
	typeOfError       = reflect.TypeOf((*error)(nil)).Elem()
	typeOfHTTPRequest = reflect.TypeOf((*http.Request)(nil)).Elem()
)

type reflector struct {
	target interface{}
}

func (r *reflector) VendorInfo() *spi.VendorInfo {
	if i, is := r.target.(spi.Vendor); is {
		return i.VendorInfo()
	}
	return nil
}

func (r *reflector) exampleProperties() *types.Any {
	if example, is := r.target.(spi.InputExample); is {
		return example.ExampleProperties()
	}
	return nil
}

// Type returns the target's type, taking into account of pointer receiver
func (r *reflector) targetType() reflect.Type {
	return reflect.Indirect(reflect.ValueOf(r.target)).Type()
}

// Interface returns the plugin type and version.
func (r *reflector) Interface() spi.InterfaceSpec {
	if v, is := r.target.(VersionedInterface); is {
		return v.ImplementedInterface()
	}
	return spi.InterfaceSpec{}
}

// isExported returns true of a string is an exported (upper case) name. -- from gorilla/rpc
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// isExportedOrBuiltin returns true if a type is exported or a builtin -- from gorilla/rpc
func isExportedOrBuiltin(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

func (r *reflector) getPluginTypeName() string {
	return r.targetType().Name()
}

func (r *reflector) setExampleProperties(param interface{}) {
	if example, is := r.target.(rpc.InputExample); is {
		example.SetExampleProperties(param)
	}
}

func (r *reflector) toDescription(m reflect.Method) plugin.MethodDescription {
	method := fmt.Sprintf("%s.%s", r.getPluginTypeName(), m.Name)
	input := reflect.New(m.Type.In(2).Elem())
	ts := fmt.Sprintf("%v", time.Now().Unix())
	d := plugin.MethodDescription{

		Request: plugin.Request{
			Version: "2.0",
			Method:  method,
			Params:  input.Interface(),
			ID:      ts,
		},

		Response: plugin.Response{
			Result: reflect.Zero(m.Type.In(3).Elem()).Interface(),
			ID:     ts,
		},
	}
	return d
}

// pluginMethods returns a slice of methods that match the criteria for exporting as RPC service
func (r *reflector) pluginMethods() []reflect.Method {
	matches := []reflect.Method{}
	receiverT := reflect.TypeOf(r.target)
	for i := 0; i < receiverT.NumMethod(); i++ {

		method := receiverT.Method(i)
		mtype := method.Type

		// Code from gorilla/rpc
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs four ins: receiver, *http.Request, *args, *reply.
		if mtype.NumIn() != 4 {
			continue
		}
		// First argument must be a pointer and must be http.Request.
		reqType := mtype.In(1)
		if reqType.Kind() != reflect.Ptr || reqType.Elem() != typeOfHTTPRequest {
			continue
		}
		// Second argument must be a pointer and must be exported.
		args := mtype.In(2)
		if args.Kind() != reflect.Ptr || !isExportedOrBuiltin(args) {
			continue
		}
		// Third argument must be a pointer and must be exported.
		reply := mtype.In(3)
		if reply.Kind() != reflect.Ptr || !isExportedOrBuiltin(reply) {
			continue
		}
		// Method needs one out: error.
		if mtype.NumOut() != 1 {
			continue
		}
		if returnType := mtype.Out(0); returnType != typeOfError {
			continue
		}

		matches = append(matches, method)
	}
	return matches
}
