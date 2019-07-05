package scw

import (
	"fmt"
	"runtime"
)

// TODO: versionning process
const version = "0.0.0"

var userAgent = fmt.Sprintf("scaleway-sdk-go/%s (%s; %s; %s)", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
