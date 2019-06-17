package scw

import (
	"fmt"
	"net/url"

	"github.com/scaleway/scaleway-sdk-go/internal/auth"
	"github.com/scaleway/scaleway-sdk-go/utils"
)

type settings struct {
	apiURL           string
	token            auth.Auth
	userAgent        string
	httpClient       httpClient
	insecure         bool
	defaultProjectID *string
	defaultRegion    *utils.Region
	defaultZone      *utils.Zone
	defaultPageSize  *int32
}

func newSettings() *settings {
	return &settings{}
}

func (s *settings) apply(opts []ClientOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *settings) validate() error {
	var err error
	if s.token == nil {
		return fmt.Errorf("no credential option provided")
	}

	_, err = url.Parse(s.apiURL)
	if err != nil {
		return fmt.Errorf("invalid url %s: %s", s.apiURL, err)
	}

	// TODO: Check ProjectID format
	if s.defaultProjectID != nil && *s.defaultProjectID == "" {
		return fmt.Errorf("default project id cannot be empty")
	}

	// TODO: Check Region format
	if s.defaultRegion != nil && *s.defaultRegion == "" {
		return fmt.Errorf("default region cannot be empty")
	}

	// TODO: Check Zone format
	if s.defaultZone != nil && *s.defaultZone == "" {
		return fmt.Errorf("default zone cannot be empty")
	}

	if s.defaultPageSize != nil && *s.defaultPageSize <= 0 {
		return fmt.Errorf("default page size cannot be <= 0")
	}

	return nil
}
