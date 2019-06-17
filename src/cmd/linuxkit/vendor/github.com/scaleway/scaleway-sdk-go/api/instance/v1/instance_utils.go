package instance

import (
	"fmt"

	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/utils"
)

// AttachIPRequest contains the parameters to attach an IP to a server
type AttachIPRequest struct {
	Zone     utils.Zone `json:"-"`
	IPID     string     `json:"-"`
	ServerID string     `json:"server_id"`
}

// AttachIPResponse contains the updated IP after attaching
type AttachIPResponse struct {
	IP *IP
}

// AttachIP attaches an IP to a server.
func (s *API) AttachIP(req *AttachIPRequest, opts ...scw.RequestOption) (*AttachIPResponse, error) {
	var ptrServerID = &req.ServerID
	ipResponse, err := s.updateIP(&updateIPRequest{
		Zone:   req.Zone,
		IPID:   req.IPID,
		Server: &ptrServerID,
	})
	if err != nil {
		return nil, err
	}

	return &AttachIPResponse{IP: ipResponse.IP}, nil
}

// DetachIPRequest contains the parameters to detach an IP from a server
type DetachIPRequest struct {
	Zone utils.Zone `json:"-"`
	IPID string     `json:"-"`
}

// DetachIPResponse contains the updated IP after detaching
type DetachIPResponse struct {
	IP *IP
}

// DetachIP detaches an IP from a server.
func (s *API) DetachIP(req *DetachIPRequest, opts ...scw.RequestOption) (*DetachIPResponse, error) {
	var ptrServerID *string
	ipResponse, err := s.updateIP(&updateIPRequest{
		Zone:   req.Zone,
		IPID:   req.IPID,
		Server: &ptrServerID,
	})
	if err != nil {
		return nil, err
	}

	return &DetachIPResponse{IP: ipResponse.IP}, nil
}

// UpdateIPRequest contains the parameters to update an IP
// if Reverse is an empty string, the reverse will be removed
type UpdateIPRequest struct {
	Zone    utils.Zone `json:"-"`
	IPID    string     `json:"-"`
	Reverse *string    `json:"reverse"`
}

// UpdateIP updates an IP
func (s *API) UpdateIP(req *UpdateIPRequest, opts ...scw.RequestOption) (*UpdateIPResponse, error) {
	var reverse **string
	if req.Reverse != nil {
		if *req.Reverse == "" {
			req.Reverse = nil
		}
		reverse = &req.Reverse
	}
	ipResponse, err := s.updateIP(&updateIPRequest{
		Zone:    req.Zone,
		IPID:    req.IPID,
		Reverse: reverse,
	})
	if err != nil {
		return nil, err
	}

	return &UpdateIPResponse{IP: ipResponse.IP}, nil
}

// AttachVolumeRequest contains the parameters to attach a volume to a server
type AttachVolumeRequest struct {
	Zone     utils.Zone `json:"-"`
	ServerID string     `json:"-"`
	VolumeID string     `json:"-"`
}

// AttachVolumeResponse contains the updated server after attaching a volume
type AttachVolumeResponse struct {
	Server *Server `json:"-"`
}

// volumesToVolumeTemplates converts a map of *Volume to a map of *VolumeTemplate
// so it can be used in a UpdateServer request
func volumesToVolumeTemplates(volumes map[string]*Volume) map[string]*VolumeTemplate {
	volumeTemplates := map[string]*VolumeTemplate{}
	for key, volume := range volumes {
		volumeTemplates[key] = &VolumeTemplate{ID: volume.ID, Name: volume.Name}
	}
	return volumeTemplates
}

// AttachVolume attaches a volume to a server
func (s *API) AttachVolume(req *AttachVolumeRequest, opts ...scw.RequestOption) (*AttachVolumeResponse, error) {
	// get server with volumes
	getServerResponse, err := s.GetServer(&GetServerRequest{
		Zone:     req.Zone,
		ServerID: req.ServerID,
	})
	if err != nil {
		return nil, err
	}
	volumes := getServerResponse.Server.Volumes

	newVolumes := volumesToVolumeTemplates(volumes)

	// add volume to volumes list
	key := fmt.Sprintf("%d", len(volumes))
	newVolumes[key] = &VolumeTemplate{
		ID: req.VolumeID,
		// name is ignored on this PATCH
		Name: req.VolumeID,
	}

	// update server
	updateServerResponse, err := s.UpdateServer(&UpdateServerRequest{
		Zone:     req.Zone,
		ServerID: req.ServerID,
		Volumes:  &newVolumes,
	})
	if err != nil {
		return nil, err
	}

	return &AttachVolumeResponse{Server: updateServerResponse.Server}, nil
}

// DetachVolumeRequest contains the parameters to detach a volume from a server
type DetachVolumeRequest struct {
	Zone     utils.Zone `json:"-"`
	VolumeID string     `json:"-"`
}

// DetachVolumeResponse contains the updated server after detaching a volume
type DetachVolumeResponse struct {
	Server *Server `json:"-"`
}

// DetachVolume detaches a volume from a server
func (s *API) DetachVolume(req *DetachVolumeRequest, opts ...scw.RequestOption) (*DetachVolumeResponse, error) {
	// get volume
	getVolumeResponse, err := s.GetVolume(&GetVolumeRequest{
		Zone:     req.Zone,
		VolumeID: req.VolumeID,
	})
	if err != nil {
		return nil, err
	}
	if getVolumeResponse.Volume == nil {
		return nil, fmt.Errorf("expected volume to have value in response")
	}
	if getVolumeResponse.Volume.Server == nil {
		return nil, fmt.Errorf("server should be attached to a server")
	}
	serverID := getVolumeResponse.Volume.Server.ID

	// get server with volumes
	getServerResponse, err := s.GetServer(&GetServerRequest{
		Zone:     req.Zone,
		ServerID: serverID,
	})
	if err != nil {
		return nil, err
	}
	volumes := getServerResponse.Server.Volumes
	// remove volume from volumes list
	for key, volume := range volumes {
		if volume.ID == req.VolumeID {
			delete(volumes, key)
		}
	}

	newVolumes := volumesToVolumeTemplates(volumes)

	// update server
	updateServerResponse, err := s.UpdateServer(&UpdateServerRequest{
		Zone:     req.Zone,
		ServerID: serverID,
		Volumes:  &newVolumes,
	})
	if err != nil {
		return nil, err
	}

	return &DetachVolumeResponse{Server: updateServerResponse.Server}, nil
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListServersResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListBootscriptsResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListIpsResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListSecurityGroupRulesResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListSecurityGroupsResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListServersTypesResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListSnapshotsResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}

// UnsafeSetTotalCount should not be used
// Internal usage only
func (r *ListVolumesResponse) UnsafeSetTotalCount(totalCount int) {
	r.TotalCount = uint32(totalCount)
}
