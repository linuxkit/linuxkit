package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
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
func (p *ProviderAWS) Probe() bool {
	// Getting the hostname should always work...
	_, err := awsGet(metaDataURL + "hostname")
	if err != nil {
		log.Printf("AWS: Probe failed: %s", strings.TrimPrefix(err.Error(), "AWS: "))
	}
	return err == nil
}

// Extract gets both the AWS specific and generic userdata
func (p *ProviderAWS) Extract() ([]byte, error) {
	// Get host name. This must not fail
	hostname, err := awsGet(metaDataURL + "hostname")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(path.Join(ConfigPath, Hostname), hostname, 0644)
	if err != nil {
		return nil, fmt.Errorf("AWS: Failed to write hostname: %s", err)
	}

	// public ipv4
	awsMetaGet("public-ipv4", "public_ipv4", 0644)

	// private ipv4
	awsMetaGet("local-ipv4", "local_ipv4", 0644)

	// availability zone
	awsMetaGet("placement/availability-zone", "availability_zone", 0644)

	// instance type
	awsMetaGet("instance-type", "instance_type", 0644)

	// instance-id
	awsMetaGet("instance-id", "instance_id", 0644)

	// local-hostname
	awsMetaGet("local-hostname", "local_hostname", 0644)

	// ssh
	if err := p.handleSSH(); err != nil {
		log.Printf("AWS: Failed to get ssh data: %s", err)
	}

	// Generic userdata
	userData, err := awsGet(userDataURL)
	if err != nil {
		log.Printf("AWS: Failed to get user-data: %s", err)
		// This is not an error
		return nil, nil
	}
	return userData, nil
}

// lookup a value (lookupName) in aws metaservice and store in given fileName
func awsMetaGet(lookupName string, fileName string, fileMode os.FileMode) {
	if lookupValue, err := awsGet(metaDataURL + lookupName); err == nil {
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

// awsTokenURL is the IMDSv2 session token endpoint (a var so tests can override it)
var awsTokenURL = "http://169.254.169.254/latest/api/token"

// IMDSv2 token state, fetched once per run: this process is one-shot, so no refresh is needed
var (
	awsTokenFetched bool
	awsToken        string
	awsTokenErr     error
)

// awsRetryDelays are the sleeps between attempts after a network-level failure
// (a var so tests can shorten it); HTTP error statuses are never retried
var awsRetryDelays = []time.Duration{1 * time.Second, 2 * time.Second}

// awsDo sends req, retrying network-level failures; req must have no body
func awsDo(client *http.Client, what string, req *http.Request) (*http.Response, error) {
	for i := 0; ; i++ {
		resp, err := client.Do(req)
		if err == nil || i == len(awsRetryDelays) {
			return resp, err
		}
		log.Printf("AWS: %s attempt %d/%d failed: %s, retrying in %s", what, i+1, len(awsRetryDelays)+1, err, awsRetryDelays[i])
		time.Sleep(awsRetryDelays[i])
	}
}

// awsFetchToken requests an IMDSv2 token. A non-200 response leaves the token
// empty (IMDSv1 fallback); no response at all means there is no metadata service.
func awsFetchToken(client *http.Client) (string, error) {
	req, err := http.NewRequest(http.MethodPut, awsTokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("AWS: http.NewRequest failed: %s", err)
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")
	resp, err := awsDo(client, "IMDSv2 token request", req)
	if err != nil {
		return "", fmt.Errorf("AWS: Could not contact metadata service: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("AWS: IMDSv2 token request returned %d, falling back to IMDSv1", resp.StatusCode)
		return "", nil
	}
	token, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("AWS: Failed to read IMDSv2 token: %s", err)
	}
	log.Printf("AWS: Using IMDSv2")
	return string(token), nil
}

// awsGet requests and extracts the requested URL
func awsGet(url string) ([]byte, error) {
	var client = &http.Client{
		Timeout: time.Second * 2,
	}

	if !awsTokenFetched {
		awsToken, awsTokenErr = awsFetchToken(client)
		awsTokenFetched = true
	}
	if awsTokenErr != nil {
		return nil, awsTokenErr
	}

	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("AWS: http.NewRequest failed: %s", err)
	}
	if awsToken != "" {
		req.Header.Set("X-aws-ec2-metadata-token", awsToken)
	}

	resp, err := awsDo(client, "GET "+url, req)
	if err != nil {
		return nil, fmt.Errorf("AWS: Could not contact metadata service: %s", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("AWS: Status not ok: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("AWS: Failed to read http response: %s", err)
	}
	return body, nil
}

// SSH keys:
func (p *ProviderAWS) handleSSH() error {
	sshKeys, err := awsGet(metaDataURL + "public-keys/0/openssh-key")
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
