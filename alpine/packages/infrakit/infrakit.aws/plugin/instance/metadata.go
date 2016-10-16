package instance

import (
	"io/ioutil"
	"net/http"
)

// MetadataKey is the identifier for a metadata entry.
type MetadataKey string

const (
	// MetadataAmiID - AMI ID
	MetadataAmiID = MetadataKey("http://169.254.169.254/latest/meta-data/ami-id")

	// MetadataInstanceID - Instance ID
	MetadataInstanceID = MetadataKey("http://169.254.169.254/latest/meta-data/instance-id")

	// MetadataInstanceType - Instance type
	MetadataInstanceType = MetadataKey("http://169.254.169.254/latest/meta-data/instance-type")

	// MetadataHostname - Host name
	MetadataHostname = MetadataKey("http://169.254.169.254/latest/meta-data/hostname")

	// MetadataLocalIPv4 - Local IPv4 address
	MetadataLocalIPv4 = MetadataKey("http://169.254.169.254/latest/meta-data/local-ipv4")

	// MetadataPublicIPv4 - Public IPv4 address
	MetadataPublicIPv4 = MetadataKey("http://169.254.169.254/latest/meta-data/public-ipv4")

	// MetadataAvailabilityZone - Availability zone
	MetadataAvailabilityZone = MetadataKey("http://169.254.169.254/latest/meta-data/placement/availability-zone")
)

// GetMetadata returns the value of the metadata by key
func GetMetadata(key MetadataKey) (string, error) {
	resp, err := http.Get(string(key))
	if err != nil {
		return "", err
	}
	buff, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	return string(buff), nil
}

// GetRegion returns the AWS region this instance is in.
func GetRegion() (string, error) {
	az, err := GetMetadata(MetadataAvailabilityZone)
	if err != nil {
		return "", err
	}
	return az[0 : len(az)-1], nil
}
