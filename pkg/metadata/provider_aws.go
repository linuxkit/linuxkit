package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
)

// ProviderAWS is the type implementing the Provider interface for AWS
type ProviderAWS struct {
}

// NewAWS returns a new ProviderAWS
func NewAWS() *ProviderAWS {
	return &ProviderAWS{}
}

func (p *ProviderAWS) String() string {
	return "AWS"
}

// Probe checks if we are running on AWS
func (p *ProviderAWS) Probe(ctx context.Context) bool {
	// Getting the hostname should always work...
	_, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
	if err != nil {
		log.Printf("AWS: Probe failed: %s", err)
	}
	return err == nil
}

// Extract gets both the AWS specific and generic userdata
func (p *ProviderAWS) Extract(ctx context.Context) ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("AWS: Failed to write hostname: %s", err)
	}

	// public ipv4
	awsMetaGet(ctx, "public-ipv4", "public_ipv4", 0644)

	// private ipv4
	awsMetaGet(ctx, "local-ipv4", "local_ipv4", 0644)

	// availability zone
	awsMetaGet(ctx, "placement/availability-zone", "availability_zone", 0644)

	// instance type
	awsMetaGet(ctx, "instance-type", "instance_type", 0644)

	// instance-id
	awsMetaGet(ctx, "instance-id", "instance_id", 0644)

	// local-hostname
	awsMetaGet(ctx, "local-hostname", "local_hostname", 0644)

	// ssh
	if err := p.handleSSH(ctx); err != nil {
		log.Printf("AWS: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := awsGet(ctx, ec2Base+"latest/user-data")
	if err != nil {
		log.Printf("AWS: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in aws metaservice and store in given fileName
func awsMetaGet(ctx context.Context, lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := awsGet(ctx, ec2Base+"latest/meta-data/"+lookupName); err == nil {
		// we got a value from the metadata server, now save to filesystem
		err = os.WriteFile(path.Join(ConfigPath, fileName), lookupValue, fileMode)
		if err != nil {
			// we couldn't save the file for some reason
			log.Printf("AWS: Failed to write %s:%s %s", fileName, lookupValue, err)
		}
	} else {
		// we did not get a value back from the metadata server
		log.Printf("AWS: Failed to get %s: %s", lookupName, err)
	}
}

// IMDSv2 token state, fetched once per run: this process is one-shot, so no
// refresh is needed. Only an HTTP answer is cached; any status other than 200
// means IMDSv1 (empty token).
var (
	awsTokenFetched bool
	awsToken        string
)

// awsGet requests and extracts the requested URL, with an IMDSv2 token if the service gives one
func awsGet(ctx context.Context, url string) ([]byte, error) {
	if !awsTokenFetched {
		hdr := http.Header{}
		hdr.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")
		token, err := fetch(ctx, http.MethodPut, ec2Base+"latest/api/token", hdr)
		var se *statusError
		switch {
		case err == nil:
			log.Printf("AWS: Using IMDSv2")
			awsToken = string(token)
		case errors.As(err, &se):
			log.Printf("AWS: IMDSv2 token request returned %d, falling back to IMDSv1", se.Code)
		default:
			return nil, err
		}
		awsTokenFetched = true
	}

	var hdr http.Header
	if awsToken != "" {
		hdr = http.Header{}
		hdr.Set("X-aws-ec2-metadata-token", awsToken)
	}
	return get(ctx, url, hdr)
}

// SSH keys:
func (p *ProviderAWS) handleSSH(ctx context.Context) error {
	sshKeys, err := awsGet(ctx, ec2Base+"latest/meta-data/public-keys/0/openssh-key")
	if err != nil {
		return fmt.Errorf("Failed to get sshKeys: %s", err)
	}

	if err := os.Mkdir(path.Join(ConfigPath, SSH), 0755); err != nil {
		return fmt.Errorf("Failed to create %s: %s", SSH, err)
	}

	err = os.WriteFile(path.Join(ConfigPath, SSH, "authorized_keys"), sshKeys, 0600)
	if err != nil {
		return fmt.Errorf("Failed to write ssh keys: %s", err)
	}
	return nil
}
