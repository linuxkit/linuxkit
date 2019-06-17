package scw

import (
	"math"
	"reflect"
	"strconv"

	"github.com/scaleway/scaleway-sdk-go/internal/errors"
)

type lister interface {
	UnsafeGetTotalCount() int
	UnsafeAppend(interface{}) (int, SdkError)
}

type legacyLister interface {
	UnsafeSetTotalCount(totalCount int)
}

// doListAll collects all pages of a List request and aggregate all results on a single response.
func (c *Client) doListAll(req *ScalewayRequest, res interface{}) (err SdkError) {

	// check for lister interface
	if response, isLister := res.(lister); isLister {

		pageCount := math.MaxUint32
		for page := 1; page <= pageCount; page++ {
			// set current page
			req.Query.Set("page", strconv.Itoa(page))

			// request the next page
			nextPage := newPage(response)
			err := c.do(req, nextPage)
			if err != nil {
				return err
			}

			// append results
			pageSize, err := response.UnsafeAppend(nextPage)
			if err != nil {
				return err
			}

			if pageSize == 0 {
				return nil
			}

			// set total count on first request
			if pageCount == math.MaxUint32 {
				totalCount := nextPage.(lister).UnsafeGetTotalCount()
				pageCount = (totalCount + pageSize - 1) / pageSize
			}
		}
		return nil
	}

	return errors.New("%T does not support pagination", res)
}

// newPage returns a variable set to the zero value of the given type
func newPage(v interface{}) interface{} {
	// reflect.New always create a pointer, that's why we use reflect.Indirect before
	return reflect.New(reflect.Indirect(reflect.ValueOf(v)).Type()).Interface()
}
