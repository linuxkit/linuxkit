package utils

// ServiceInfo contains API metadata
// These metadata are only here for debugging. Do not rely on these values
type ServiceInfo struct {
	// Name is the name of the API
	Name string `json:"name"`

	// Description is a human readable description for the API
	Description string `json:"description"`

	// Version is the version of the API
	Version string `json:"version"`

	// DocumentationUrl is the a web url where the documentation of the API can be found
	DocumentationUrl *string `json:"documentation_url"`
}
