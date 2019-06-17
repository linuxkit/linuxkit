// This file was automatically generated. DO NOT EDIT.
// If you have any remark or suggestion do not hesitate to open an issue.

package marketplace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/scaleway/scaleway-sdk-go/internal/errors"
	"github.com/scaleway/scaleway-sdk-go/internal/marshaler"
	"github.com/scaleway/scaleway-sdk-go/internal/parameter"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/utils"
)

// always import dependencies
var (
	_ fmt.Stringer
	_ json.Unmarshaler
	_ url.URL
	_ net.IP
	_ http.Header
	_ bytes.Reader
	_ time.Time

	_ scw.ScalewayRequest
	_ marshaler.Duration
	_ utils.File
	_ = parameter.AddToQuery
)

// API marketplace API
type API struct {
	client *scw.Client
}

// NewAPI returns a API object from a Scaleway client.
func NewAPI(client *scw.Client) *API {
	return &API{
		client: client,
	}
}

type GetImageResponse struct {
	Image *Image `json:"image,omitempty"`
}

type GetServiceInfoResponse struct {
	API string `json:"api,omitempty"`

	Description string `json:"description,omitempty"`

	Version string `json:"version,omitempty"`
}

type GetVersionResponse struct {
	Version *Version `json:"version,omitempty"`
}

type Image struct {
	ID string `json:"id,omitempty"`

	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`

	Logo string `json:"logo,omitempty"`

	Categories []string `json:"categories,omitempty"`

	Organization *Organization `json:"organization,omitempty"`

	ValidUntil time.Time `json:"valid_until,omitempty"`

	CreationDate time.Time `json:"creation_date,omitempty"`

	ModificationDate time.Time `json:"modification_date,omitempty"`

	Versions []*Version `json:"versions,omitempty"`

	CurrentPublicVersion string `json:"current_public_version,omitempty"`
}

type ListImagesResponse struct {
	Images []*Image `json:"images,omitempty"`
}

type ListVersionsResponse struct {
	Versions []*Version `json:"versions,omitempty"`
}

type LocalImage struct {
	ID string `json:"id,omitempty"`

	Arch string `json:"arch,omitempty"`

	Zone utils.Zone `json:"zone,omitempty"`

	CompatibleCommercialTypes []string `json:"compatible_commercial_types,omitempty"`
}

type Organization struct {
	ID string `json:"id,omitempty"`

	Name string `json:"name,omitempty"`
}

type Version struct {
	ID string `json:"id,omitempty"`

	Name string `json:"name,omitempty"`

	CreationDate time.Time `json:"creation_date,omitempty"`

	ModificationDate time.Time `json:"modification_date,omitempty"`

	LocalImages []*LocalImage `json:"local_images,omitempty"`
}

// Service API

type GetServiceInfoRequest struct {
}

func (s *API) GetServiceInfo(req *GetServiceInfoRequest, opts ...scw.RequestOption) (*GetServiceInfoResponse, error) {
	var err error

	scwReq := &scw.ScalewayRequest{
		Method:  "GET",
		Path:    "/marketplace/v1",
		Headers: http.Header{},
	}

	var resp GetServiceInfoResponse

	err = s.client.Do(scwReq, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type ListImagesRequest struct {
	PerPage *int32 `json:"-"`

	Page *int32 `json:"-"`
}

func (s *API) ListImages(req *ListImagesRequest, opts ...scw.RequestOption) (*ListImagesResponse, error) {
	var err error

	defaultPerPage, exist := s.client.GetDefaultPageSize()
	if (req.PerPage == nil || *req.PerPage == 0) && exist {
		req.PerPage = &defaultPerPage
	}

	query := url.Values{}
	parameter.AddToQuery(query, "per_page", req.PerPage)
	parameter.AddToQuery(query, "page", req.Page)

	scwReq := &scw.ScalewayRequest{
		Method:  "GET",
		Path:    "/marketplace/v1/images",
		Query:   query,
		Headers: http.Header{},
	}

	var resp ListImagesResponse

	err = s.client.Do(scwReq, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type GetImageRequest struct {
	ImageID string `json:"-"`
}

func (s *API) GetImage(req *GetImageRequest, opts ...scw.RequestOption) (*GetImageResponse, error) {
	var err error

	if fmt.Sprint(req.ImageID) == "" {
		return nil, errors.New("field ImageID cannot be empty in request")
	}

	scwReq := &scw.ScalewayRequest{
		Method:  "GET",
		Path:    "/marketplace/v1/images/" + fmt.Sprint(req.ImageID) + "",
		Headers: http.Header{},
	}

	var resp GetImageResponse

	err = s.client.Do(scwReq, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type ListVersionsRequest struct {
	ImageID string `json:"-"`
}

func (s *API) ListVersions(req *ListVersionsRequest, opts ...scw.RequestOption) (*ListVersionsResponse, error) {
	var err error

	if fmt.Sprint(req.ImageID) == "" {
		return nil, errors.New("field ImageID cannot be empty in request")
	}

	scwReq := &scw.ScalewayRequest{
		Method:  "GET",
		Path:    "/marketplace/v1/images/" + fmt.Sprint(req.ImageID) + "/versions",
		Headers: http.Header{},
	}

	var resp ListVersionsResponse

	err = s.client.Do(scwReq, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type GetVersionRequest struct {
	ImageID string `json:"-"`

	VersionID string `json:"-"`
}

func (s *API) GetVersion(req *GetVersionRequest, opts ...scw.RequestOption) (*GetVersionResponse, error) {
	var err error

	if fmt.Sprint(req.ImageID) == "" {
		return nil, errors.New("field ImageID cannot be empty in request")
	}

	if fmt.Sprint(req.VersionID) == "" {
		return nil, errors.New("field VersionID cannot be empty in request")
	}

	scwReq := &scw.ScalewayRequest{
		Method:  "GET",
		Path:    "/marketplace/v1/images/" + fmt.Sprint(req.ImageID) + "/versions/" + fmt.Sprint(req.VersionID) + "",
		Headers: http.Header{},
	}

	var resp GetVersionResponse

	err = s.client.Do(scwReq, &resp, opts...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
