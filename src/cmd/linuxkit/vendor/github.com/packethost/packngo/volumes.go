package packngo

import "fmt"

const (
	volumeBasePath      = "/storage"
	attachmentsBasePath = "/attachments"
)

// VolumeService interface defines available Volume methods
type VolumeService interface {
	Get(string) (*Volume, *Response, error)
	Update(*VolumeUpdateRequest) (*Volume, *Response, error)
	Delete(string) (*Response, error)
	Create(*VolumeCreateRequest, string) (*Volume, *Response, error)
}

// VolumeAttachmentService defines attachment methdods
type VolumeAttachmentService interface {
	Get(string) (*VolumeAttachment, *Response, error)
	Create(string, string) (*VolumeAttachment, *Response, error)
	Delete(string) (*Response, error)
}

// Volume represents a volume
type Volume struct {
	ID               string              `json:"id"`
	Name             string              `json:"name,omitempty"`
	Description      string              `json:"description,omitempty"`
	Size             int                 `json:"size,omitempty"`
	State            string              `json:"state,omitempty"`
	Locked           bool                `json:"locked,omitempty"`
	BillingCycle     string              `json:"billing_cycle,omitempty"`
	Created          string              `json:"created_at,omitempty"`
	Updated          string              `json:"updated_at,omitempty"`
	Href             string              `json:"href,omitempty"`
	SnapshotPolicies []*SnapshotPolicy   `json:"snapshot_policies,omitempty"`
	Attachments      []*VolumeAttachment `json:"attachments,omitempty"`
	Plan             *Plan               `json:"plan,omitempty"`
	Facility         *Facility           `json:"facility,omitempty"`
	Project          *Project            `json:"project,omitempty"`
}

// SnapshotPolicy used to execute actions on volume
type SnapshotPolicy struct {
	ID                string `json:"id"`
	Href              string `json:"href"`
	SnapshotFrequency string `json:"snapshot_frequency,omitempty"`
	SnapshotCount     int    `json:"snapshot_count,omitempty"`
}

func (v Volume) String() string {
	return Stringify(v)
}

// VolumeCreateRequest type used to create a Packet volume
type VolumeCreateRequest struct {
	Size             int               `json:"size"`
	BillingCycle     string            `json:"billing_cycle"`
	ProjectID        string            `json:"project_id"`
	PlanID           string            `json:"plan_id"`
	FacilityID       string            `json:"facility_id"`
	Description      string            `json:"description,omitempty"`
	SnapshotPolicies []*SnapshotPolicy `json:"snapshot_policies,omitempty"`
}

func (v VolumeCreateRequest) String() string {
	return Stringify(v)
}

// VolumeUpdateRequest type used to update a Packet volume
type VolumeUpdateRequest struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	Plan        string `json:"plan,omitempty"`
}

// VolumeAttachment is a type from Packet API
type VolumeAttachment struct {
	Href   string `json:"href"`
	ID     string `json:"id"`
	Volume Volume `json:"volume"`
	Device Device `json:"device"`
}

func (v VolumeUpdateRequest) String() string {
	return Stringify(v)
}

// VolumeAttachmentServiceOp implements VolumeService
type VolumeAttachmentServiceOp struct {
	client *Client
}

// VolumeServiceOp implements VolumeService
type VolumeServiceOp struct {
	client *Client
}

// Get returns a volume by id
func (v *VolumeServiceOp) Get(volumeID string) (*Volume, *Response, error) {
	path := fmt.Sprintf("%s/%s?include=facility,snapshot_policies,attachments.device", volumeBasePath, volumeID)
	volume := new(Volume)

	resp, err := v.client.DoRequest("GET", path, nil, volume)
	if err != nil {
		return nil, resp, err
	}

	return volume, resp, err
}

// Update updates a volume
func (v *VolumeServiceOp) Update(updateRequest *VolumeUpdateRequest) (*Volume, *Response, error) {
	path := fmt.Sprintf("%s/%s", volumeBasePath, updateRequest.ID)
	volume := new(Volume)

	resp, err := v.client.DoRequest("PATCH", path, updateRequest, volume)
	if err != nil {
		return nil, resp, err
	}

	return volume, resp, err
}

// Delete deletes a volume
func (v *VolumeServiceOp) Delete(volumeID string) (*Response, error) {
	path := fmt.Sprintf("%s/%s", volumeBasePath, volumeID)

	return v.client.DoRequest("DELETE", path, nil, nil)
}

// Create creates a new volume for a project
func (v *VolumeServiceOp) Create(createRequest *VolumeCreateRequest, projectID string) (*Volume, *Response, error) {
	url := fmt.Sprintf("%s/%s%s", projectBasePath, projectID, volumeBasePath)
	volume := new(Volume)

	resp, err := v.client.DoRequest("POST", url, createRequest, volume)
	if err != nil {
		return nil, resp, err
	}

	return volume, resp, err
}

// Attachments

// Create Attachment, i.e. attach volume to a device
func (v *VolumeAttachmentServiceOp) Create(volumeID, deviceID string) (*VolumeAttachment, *Response, error) {
	url := fmt.Sprintf("%s/%s%s", volumeBasePath, volumeID, attachmentsBasePath)
	volAttachParam := map[string]string{
		"device_id": deviceID,
	}
	volumeAttachment := new(VolumeAttachment)

	resp, err := v.client.DoRequest("POST", url, volAttachParam, volumeAttachment)
	if err != nil {
		return nil, resp, err
	}
	return volumeAttachment, resp, nil
}

// Get gets attachment by id
func (v *VolumeAttachmentServiceOp) Get(attachmentID string) (*VolumeAttachment, *Response, error) {
	path := fmt.Sprintf("%s%s/%s", volumeBasePath, attachmentsBasePath, attachmentID)
	volumeAttachment := new(VolumeAttachment)

	resp, err := v.client.DoRequest("GET", path, nil, volumeAttachment)
	if err != nil {
		return nil, resp, err
	}

	return volumeAttachment, resp, nil
}

// Delete deletes attachment by id
func (v *VolumeAttachmentServiceOp) Delete(attachmentID string) (*Response, error) {
	path := fmt.Sprintf("%s%s/%s", volumeBasePath, attachmentsBasePath, attachmentID)

	return v.client.DoRequest("DELETE", path, nil, nil)
}
