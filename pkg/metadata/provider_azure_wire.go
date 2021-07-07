package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const (
	wireServerURL string = "http://168.63.129.16/machine"
)

// WireServerClient used to report ready to Azure
//  1. GET Goal State from WireServer.
//  2. Build XML repsonse, by extracing ContainerId and InstanceId from
//     goal state.
//  3. POST XML response to WireServer indicating successful provisioning.
// See also:
//  https://docs.microsoft.com/en-us/azure/virtual-machines/linux/no-agent#generic-steps-without-using-python
type WireServerClient struct{}

// GoalState XML model (request)
type GoalState struct {
	ContainerID string `xml:"Container>ContainerId"`
	InstanceID  string `xml:"Container>RoleInstanceList>RoleInstance>InstanceId"`
}

// Health XML model (response)
type Health struct {
	GoalStateIncarnation string `xml:"GoalStateIncarnation"`
	ContainerID          string `xml:"Container>ContainerId"`
	InstanceID           string `xml:"Container>RoleInstanceList>Role>InstanceId"`
	State                string `xml:"Container>RoleInstanceList>Role>Health>State"`
}

// ReportReady to Azure's WireServer, indicating successful provisioning
func ReportReady(client *http.Client) error {
	goalState, err := getGoalState(client)
	if err != nil {
		return fmt.Errorf("Report ready: GET goal state: %s", err)
	}
	reportReadyXML, err := buildXML(goalState.ContainerID, goalState.InstanceID)
	if err != nil {
		return fmt.Errorf("Report ready: Build XML: %s", err)
	}
	err = postReportReady(client, reportReadyXML)
	if err != nil {
		return fmt.Errorf("Report ready: POST XML: %s", err)
	}
	log.Debugf(
		"Report ready: ContainerId=%s InstanceId=%s succeeded",
		goalState.ContainerID,
		goalState.InstanceID,
	)
	return nil
}

// Get goal state from WireServer
func getGoalState(client *http.Client) (*GoalState, error) {
	req, err := http.NewRequest("GET", wireServerURL+"?comp=goalstate", nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest failed: %s", err)
	}
	req.Header.Set("x-ms-version", "2012-11-30")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("WireServer unavailable: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("WireServer returned status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Reading HTTP response failed: %s", err)
	}

	var goalState GoalState
	if err = xml.Unmarshal(body, &goalState); err != nil {
		return nil, fmt.Errorf("Unmarshalling XML failed: %s", err)
	}
	return &goalState, nil
}

// Build report ready XML from container and instance ID
func buildXML(containerID, instanceID string) ([]byte, error) {
	xmlBytes, err := xml.Marshal(
		Health{
			GoalStateIncarnation: "1",
			ContainerID:          containerID,
			InstanceID:           instanceID,
			State:                "Ready"})
	if err != nil {
		return nil, fmt.Errorf("Marshalling XML failed: %s", err)
	}
	return xmlBytes, nil
}

// Post report ready XML to WireServer
func postReportReady(client *http.Client, body []byte) error {
	req, err := http.NewRequest("POST", wireServerURL+"?comp=health", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("http.NewRequest failed: %s", err)
	}
	req.Header.Set("x-ms-version", "2012-11-30")
	req.Header.Set("x-ms-agent-name", "WALinuxAgent")
	req.Header.Set("Content-Type", "text/xml;charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("WireServer unavailable: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("WireServer returned status code: %d", resp.StatusCode)
	}
	return err
}
