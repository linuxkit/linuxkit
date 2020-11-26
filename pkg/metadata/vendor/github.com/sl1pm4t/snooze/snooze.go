package snooze

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strings"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"
)

type Client struct {
	Before      func(*retryablehttp.Request, *retryablehttp.Client)
	HandleError func(*ErrorResponse) error
	Root        string
	Logger      *logrus.Logger
}

type ErrorResponse struct {
	Status              string
	StatusCode          int
	ResponseBody        []byte
	ResponseContentType string
}

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("%s [%s]", e.Status, e.ResponseContentType)
}

type resultInfo struct {
	errorIndex          int
	payloadIndex        int
	payloadType         reflect.Type
	resultLength        int
	responseContentType string
}

func (info *resultInfo) result(err error, bytes []byte) []reflect.Value {
	result := make([]reflect.Value, info.resultLength)
	if info.errorIndex > -1 {
		if err != nil {
			result[info.errorIndex] = reflect.ValueOf(&err).Elem()
		} else {
			result[info.errorIndex] = nilError
		}
	}
	if info.payloadIndex > -1 {
		if bytes != nil {
			target := reflect.New(info.payloadType)

			switch info.payloadType.Name() {
			case "string":
				contents := string(bytes)
				result[info.payloadIndex] = reflect.ValueOf(contents)

			default:
				respContentType := info.responseContentType
				if respContentType != "" {
					if strings.Contains(respContentType, ";") {
						// strip any extra detail
						respContentType = respContentType[:strings.Index(respContentType, ";")]
					}
				} else {
					respContentType = "application/json"
				}
				switch respContentType {
				case "application/json":
					err = json.Unmarshal(bytes, target.Interface())
				case "application/xml":
					err = xml.Unmarshal(bytes, target.Interface())
				case "text/xml":
					err = xml.Unmarshal(bytes, target.Interface())
				default:
					fmt.Printf("\nContent Type (%s) not supported by snooze.\n", respContentType)
				}

				if err != nil {
					return info.result(err, nil)
				}
				result[info.payloadIndex] = target.Elem()
			}

		} else {
			result[info.payloadIndex] = reflect.Zero(info.payloadType)
		}
	}
	return result
}

var nilError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())

func (c *Client) Create(in interface{}) {
	inputValue := reflect.ValueOf(in).Elem()
	inputType := inputValue.Type()

	for i := 0; i < inputValue.NumField(); i++ {

		fieldValue := inputValue.Field(i)
		fieldStruct := inputType.Field(i)
		fieldType := fieldStruct.Type
		originalPath := fieldStruct.Tag.Get("path")
		method := fieldStruct.Tag.Get("method")
		contentType := fieldStruct.Tag.Get("contentType")

		if contentType == "" {
			contentType = "application/json"
		}
		var body interface{}

		info := resultInfo{
			resultLength: fieldType.NumOut(),
			errorIndex:   -1,
			payloadIndex: -1,
		}

		for n := 0; n < info.resultLength; n++ {
			out := fieldType.Out(n)
			if out == reflect.TypeOf((*error)(nil)).Elem() {
				info.errorIndex = n
			} else {
				info.payloadIndex = n
				info.payloadType = out
			}
		}

		fieldValue.Set(reflect.MakeFunc(fieldType, func(args []reflect.Value) []reflect.Value {
			// Prepare Request Parameters
			path := originalPath
			for n, av := range args {
				if av.Kind() == reflect.Struct || av.Kind() == reflect.Ptr {
					body = av.Interface()
					continue
				}
				path = strings.Replace(path, fmt.Sprintf("{%v}", n), url.QueryEscape(fmt.Sprint(av.Interface())), -1)
			}

			// Prepare Request Body
			var err error
			buffer := make([]byte, 0)
			if method != "GET" && body != nil {

				switch contentType {
				case "application/json":
					buffer, err = json.Marshal(body)
				case "application/xml":
					buffer, err = xml.Marshal(body)
				default:
					return info.result(fmt.Errorf("ContentType (%s) not supported.", contentType), nil)
				}
				if err != nil {
					return info.result(err, nil)
				}
			}

			// Prepare Request
			req, err := retryablehttp.NewRequest(method, c.Root+path, bytes.NewReader(buffer))
			if err != nil {
				return info.result(err, nil)
			}
			req.Header.Set("Content-Type", contentType)
			client := retryablehttp.NewClient()
			if c.Before != nil {
				c.Before(req, client)
			}

			if c.Logger != nil {
				dump, _ := httputil.DumpRequest(req.Request, true)
				reqdump := strings.Replace(string(dump), "\\n", "\n", -1)
				c.Logger.Debugf("REQUEST --->\n%q\n", reqdump)
			}

			// Send Request
			resp, err := client.Do(req)
			if err != nil {
				return info.result(err, nil)
			}

			if c.Logger != nil {
				dump, _ := httputil.DumpResponse(resp, true)
				respdump := strings.Replace(string(dump), "\\n", "\n", -1)
				c.Logger.Debugf("RESPONSE <---\n%q\n", respdump)
			}

			// Process Response
			info.responseContentType = resp.Header.Get("Content-Type")
			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return info.result(err, nil)
			}
			if isErrorResponse(resp) {
				apiErr := ErrorResponse{
					Status:              resp.Status,
					StatusCode:          resp.StatusCode,
					ResponseBody:        bytes,
					ResponseContentType: info.responseContentType,
				}
				var handled error
				if c.HandleError != nil {
					handled = c.HandleError(&apiErr)
				} else {
					handled = apiErr
				}

				return info.result(handled, nil)
			} else {
				return info.result(nil, bytes)
			}
		}))
	}
}

func isErrorResponse(r *http.Response) bool {
	if r.StatusCode > 399 {
		return true
	}

	return false
}
