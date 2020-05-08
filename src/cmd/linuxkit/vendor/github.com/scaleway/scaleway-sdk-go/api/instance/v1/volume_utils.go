package instance

import (
	"fmt"
	"net/http"
	"time"

	"github.com/scaleway/scaleway-sdk-go/internal/errors"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// UpdateVolumeRequest contains the parameters to update on a volume
type UpdateVolumeRequest struct {
	Zone scw.Zone `json:"-"`
	// VolumeID is the volumes unique ID
	VolumeID string `json:"-"`
	// Name display the volumes names
	Name *string `json:"name,omitempty"`
}

// UpdateVolumeResponse contains the updated volume.
type UpdateVolumeResponse struct {
	Volume *Volume `json:"volume,omitempty"`
}

// setVolumeRequest contains all the params to PUT volumes
type setVolumeRequest struct {
	Zone scw.Zone `json:"-"`
	// ID display the volumes unique ID
	ID string `json:"id"`
	// Name display the volumes names
	Name string `json:"name"`
	// ExportURI show the volumes NBD export URI
	ExportURI string `json:"export_uri"`
	// Size display the volumes disk size
	Size scw.Size `json:"size"`
	// VolumeType display the volumes type
	//
	// Default value: l_ssd
	VolumeType VolumeType `json:"volume_type"`
	// CreationDate display the volumes creation date
	CreationDate time.Time `json:"creation_date"`
	// ModificationDate display the volumes modification date
	ModificationDate time.Time `json:"modification_date"`
	// Organization display the volumes organization
	Organization string `json:"organization"`
	// Server display information about the server attached to the volume
	Server *ServerSummary `json:"server"`
}

// UpdateVolume updates the set fields on the volume.
func (s *API) UpdateVolume(req *UpdateVolumeRequest, opts ...scw.RequestOption) (*UpdateVolumeResponse, error) {
	var err error

	if req.Zone == "" {
		defaultZone, _ := s.client.GetDefaultZone()
		req.Zone = defaultZone
	}

	if fmt.Sprint(req.Zone) == "" {
		return nil, errors.New("field Zone cannot be empty in request")
	}

	if fmt.Sprint(req.VolumeID) == "" {
		return nil, errors.New("field VolumeID cannot be empty in request")
	}

	getVolumeResponse, err := s.GetVolume(&GetVolumeRequest{
		Zone:     req.Zone,
		VolumeID: req.VolumeID,
	})
	if err != nil {
		return nil, err
	}

	setVolumeRequest := &setVolumeRequest{
		Zone:             req.Zone,
		ID:               getVolumeResponse.Volume.ID,
		Name:             getVolumeResponse.Volume.Name,
		ExportURI:        getVolumeResponse.Volume.ExportURI,
		Size:             getVolumeResponse.Volume.Size,
		VolumeType:       getVolumeResponse.Volume.VolumeType,
		CreationDate:     getVolumeResponse.Volume.CreationDate,
		ModificationDate: getVolumeResponse.Volume.ModificationDate,
		Organization:     getVolumeResponse.Volume.Organization,
		Server:           getVolumeResponse.Volume.Server,
	}

	// Override the values that need to be updated
	if req.Name != nil {
		setVolumeRequest.Name = *req.Name
	}

	scwReq := &scw.ScalewayRequest{
		Method:  "PUT",
		Path:    "/instance/v1/zones/" + fmt.Sprint(req.Zone) + "/volumes/" + fmt.Sprint(req.VolumeID) + "",
		Headers: http.Header{},
	}

	err = scwReq.SetBody(setVolumeRequest)
	if err != nil {
		return nil, err
	}

	var res UpdateVolumeResponse

	err = s.client.Do(scwReq, &res, opts...)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
