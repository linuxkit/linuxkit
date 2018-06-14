// Copyright (C) 2018 Scaleway. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

// Interact with Scaleway API

// Package api contains client and functions to interact with Scaleway API
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/scaleway/go-scaleway/cache"
	"github.com/scaleway/go-scaleway/logger"
	"github.com/scaleway/go-scaleway/types"
)

// Default values
var (
	AccountAPI     = "https://account.scaleway.com/"
	MetadataAPI    = "http://169.254.42.42/"
	MarketplaceAPI = "https://api-marketplace.scaleway.com"
	ComputeAPIPar1 = "https://cp-par1.scaleway.com/"
	ComputeAPIAms1 = "https://cp-ams1.scaleway.com"

	URLPublicDNS  = ".pub.cloud.scaleway.com"
	URLPrivateDNS = ".priv.cloud.scaleway.com"
)

func init() {
	if url := os.Getenv("SCW_ACCOUNT_API"); url != "" {
		AccountAPI = url
	}
	if url := os.Getenv("SCW_METADATA_API"); url != "" {
		MetadataAPI = url
	}
	if url := os.Getenv("SCW_MARKETPLACE_API"); url != "" {
		MarketplaceAPI = url
	}
	if url := os.Getenv("SCW_COMPUTE_PAR1_API"); url != "" {
		ComputeAPIPar1 = url
	}
	if url := os.Getenv("SCW_COMPUTE_AMS1_API"); url != "" {
		ComputeAPIAms1 = url
	}
}

const (
	perPage = 50
)

// types.ScalewayAPI is the interface used to communicate with the Scaleway API
type ScalewayAPI struct {
	// Organization is the identifier of the Scaleway organization
	Organization string

	// Token is the authentication token for the Scaleway organization
	Token string

	// Password is the authentication password
	password string

	userAgent string

	// Cache is used to quickly resolve identifiers from names
	Cache *cache.ScalewayCache

	client     *http.Client
	verbose    bool
	computeAPI string

	Region string
	//
	logger.Logger
}

// Newtypes.ScalewayAPI creates a ready-to-use ScalewayAPI client
func NewScalewayAPI(organization, token, userAgent, region string, options ...func(*ScalewayAPI)) (*ScalewayAPI, error) {
	s := &ScalewayAPI{
		// exposed
		Organization: organization,
		Token:        token,
		Logger:       logger.NewDefaultLogger(),

		// internal
		client:    &http.Client{},
		verbose:   os.Getenv("SCW_VERBOSE_API") != "",
		password:  "",
		userAgent: userAgent,
	}
	for _, option := range options {
		option(s)
	}
	cache, err := cache.NewScalewayCache(func() { s.Logger.Debugf("Writing cache file to disk") })
	if err != nil {
		return nil, err
	}
	s.Cache = cache
	if os.Getenv("SCW_TLSVERIFY") == "0" {
		s.client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	switch region {
	case "par1", "":
		s.computeAPI = ComputeAPIPar1
	case "ams1":
		s.computeAPI = ComputeAPIAms1
	default:
		return nil, fmt.Errorf("%s isn't a valid region", region)
	}
	s.Region = region
	if url := os.Getenv("SCW_COMPUTE_API"); url != "" {
		s.computeAPI = url
	}
	return s, nil
}

// ClearCache clears the cache
func (s *ScalewayAPI) ClearCache() {
	s.Cache.Clear()
}

// Sync flushes out the cache to the disk
func (s *ScalewayAPI) Sync() {
	s.Cache.Save()
}

func (s *ScalewayAPI) response(method, uri string, content io.Reader) (resp *http.Response, err error) {
	var (
		req *http.Request
	)

	req, err = http.NewRequest(method, uri, content)
	if err != nil {
		err = fmt.Errorf("response %s %s", method, uri)
		return
	}
	req.Header.Set("X-Auth-Token", s.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.userAgent)
	s.LogHTTP(req)
	if s.verbose {
		dump, _ := httputil.DumpRequest(req, true)
		s.Debugf("%v", string(dump))
	} else {
		s.Debugf("[%s]: %v", method, uri)
	}
	resp, err = s.client.Do(req)
	return
}

// GetResponsePaginate fetchs all resources and returns an http.Response object for the requested resource
func (s *ScalewayAPI) GetResponsePaginate(apiURL, resource string, values url.Values) (*http.Response, error) {
	resp, err := s.response("HEAD", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, values.Encode()), nil)
	if err != nil {
		return nil, err
	}

	count := resp.Header.Get("X-Total-Count")
	var maxElem int
	if count == "" {
		maxElem = 0
	} else {
		maxElem, err = strconv.Atoi(count)
		if err != nil {
			return nil, err
		}
	}

	get := maxElem / perPage
	if (float32(maxElem) / perPage) > float32(get) {
		get++
	}

	if get <= 1 { // If there is 0 or 1 page of result, the response is not paginated
		if len(values) == 0 {
			return s.response("GET", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), nil)
		}
		return s.response("GET", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, values.Encode()), nil)
	}

	fetchAll := !(values.Get("per_page") != "" || values.Get("page") != "")
	if fetchAll {
		var g errgroup.Group

		ch := make(chan *http.Response, get)
		for i := 1; i <= get; i++ {
			i := i // closure tricks
			g.Go(func() (err error) {
				var resp *http.Response

				val := url.Values{}
				val.Set("per_page", fmt.Sprintf("%v", perPage))
				val.Set("page", fmt.Sprintf("%v", i))
				resp, err = s.response("GET", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, val.Encode()), nil)
				ch <- resp
				return
			})
		}
		if err = g.Wait(); err != nil {
			return nil, err
		}
		newBody := make(map[string][]json.RawMessage)
		body := make(map[string][]json.RawMessage)
		key := ""
		for i := 0; i < get; i++ {
			res := <-ch
			if res.StatusCode != http.StatusOK {
				return res, nil
			}
			content, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(content, &body); err != nil {
				return nil, err
			}

			if i == 0 {
				resp = res
				for k := range body {
					key = k
					break
				}
			}
			newBody[key] = append(newBody[key], body[key]...)
		}
		payload := new(bytes.Buffer)
		if err := json.NewEncoder(payload).Encode(newBody); err != nil {
			return nil, err
		}
		resp.Body = ioutil.NopCloser(payload)
	} else {
		resp, err = s.response("GET", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, values.Encode()), nil)
	}
	return resp, err
}

// PostResponse returns an http.Response object for the updated resource
func (s *ScalewayAPI) PostResponse(apiURL, resource string, data interface{}) (*http.Response, error) {
	payload := new(bytes.Buffer)
	if err := json.NewEncoder(payload).Encode(data); err != nil {
		return nil, err
	}
	return s.response("POST", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), payload)
}

// PatchResponse returns an http.Response object for the updated resource
func (s *ScalewayAPI) PatchResponse(apiURL, resource string, data interface{}) (*http.Response, error) {
	payload := new(bytes.Buffer)
	if err := json.NewEncoder(payload).Encode(data); err != nil {
		return nil, err
	}
	return s.response("PATCH", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), payload)
}

// PutResponse returns an http.Response object for the updated resource
func (s *ScalewayAPI) PutResponse(apiURL, resource string, data interface{}) (*http.Response, error) {
	payload := new(bytes.Buffer)
	if err := json.NewEncoder(payload).Encode(data); err != nil {
		return nil, err
	}
	return s.response("PUT", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), payload)
}

// DeleteResponse returns an http.Response object for the deleted resource
func (s *ScalewayAPI) DeleteResponse(apiURL, resource string) (*http.Response, error) {
	return s.response("DELETE", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), nil)
}

// handleHTTPError checks the statusCode and displays the error
func (s *ScalewayAPI) handleHTTPError(goodStatusCode []int, resp *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if s.verbose {
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			var js bytes.Buffer

			err = json.Indent(&js, body, "", "  ")
			if err != nil {
				s.Debugf("[Response]: [%v]\n%v", resp.StatusCode, string(dump))
			} else {
				s.Debugf("[Response]: [%v]\n%v", resp.StatusCode, js.String())
			}
		}
	} else {
		s.Debugf("[Response]: [%v]\n%v", resp.StatusCode, string(body))
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, errors.New(string(body))
	}
	good := false
	for _, code := range goodStatusCode {
		if code == resp.StatusCode {
			good = true
		}
	}
	if !good {
		var scwError types.ScalewayAPIError

		if err := json.Unmarshal(body, &scwError); err != nil {
			return nil, err
		}
		scwError.StatusCode = resp.StatusCode
		s.Debugf("%s", scwError.Error())
		return nil, scwError
	}
	return body, nil
}

func (s *ScalewayAPI) fetchServers(api string, query url.Values, out chan<- types.ScalewayServers) func() error {
	return func() error {
		resp, err := s.GetResponsePaginate(api, "servers", query)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
		if err != nil {
			return err
		}
		var servers types.ScalewayServers

		if err = json.Unmarshal(body, &servers); err != nil {
			return err
		}
		out <- servers
		return nil
	}
}

// GetServers gets the list of servers from the ScalewayAPI
func (s *ScalewayAPI) GetServers(all bool, limit int) (*[]types.ScalewayServer, error) {
	query := url.Values{}
	if !all {
		query.Set("state", "running")
	}
	if limit > 0 {
		// FIXME: wait for the API to be ready
		// query.Set("per_page", strconv.Itoa(limit))
		panic("Not implemented yet")
	}
	if all && limit == 0 {
		s.Cache.ClearServers()
	}
	var (
		g    errgroup.Group
		apis = []string{
			ComputeAPIPar1,
			ComputeAPIAms1,
		}
	)

	serverChan := make(chan types.ScalewayServers, 2)
	for _, api := range apis {
		g.Go(s.fetchServers(api, query, serverChan))
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(serverChan)
	var servers types.ScalewayServers

	for server := range serverChan {
		servers.Servers = append(servers.Servers, server.Servers...)
	}

	for i, server := range servers.Servers {
		servers.Servers[i].DNSPublic = server.Identifier + URLPublicDNS
		servers.Servers[i].DNSPrivate = server.Identifier + URLPrivateDNS
		s.Cache.InsertServer(server.Identifier, server.Location.ZoneID, server.Arch, server.Organization, server.Name)
	}
	return &servers.Servers, nil
}

// GetServer gets a server from the ScalewayAPI
func (s *ScalewayAPI) GetServer(serverID string) (*types.ScalewayServer, error) {
	if serverID == "" {
		return nil, fmt.Errorf("cannot get server without serverID")
	}
	resp, err := s.GetResponsePaginate(s.computeAPI, "servers/"+serverID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}

	var oneServer types.ScalewayOneServer

	if err = json.Unmarshal(body, &oneServer); err != nil {
		return nil, err
	}
	// FIXME arch, owner, title
	oneServer.Server.DNSPublic = oneServer.Server.Identifier + URLPublicDNS
	oneServer.Server.DNSPrivate = oneServer.Server.Identifier + URLPrivateDNS
	s.Cache.InsertServer(oneServer.Server.Identifier, oneServer.Server.Location.ZoneID, oneServer.Server.Arch, oneServer.Server.Organization, oneServer.Server.Name)
	return &oneServer.Server, nil
}

// PostServerAction posts an action on a server
func (s *ScalewayAPI) PostServerAction(serverID, action string) error {
	data := types.ScalewayServerAction{
		Action: action,
	}
	resp, err := s.PostResponse(s.computeAPI, fmt.Sprintf("servers/%s/action", serverID), data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// DeleteServer deletes a server
func (s *ScalewayAPI) DeleteServer(serverID string) error {
	defer s.Cache.RemoveServer(serverID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("servers/%s", serverID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err = s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// PostServer creates a new server
func (s *ScalewayAPI) PostServer(definition types.ScalewayServerDefinition) (string, error) {
	definition.Organization = s.Organization

	resp, err := s.PostResponse(s.computeAPI, "servers", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var server types.ScalewayOneServer

	if err = json.Unmarshal(body, &server); err != nil {
		return "", err
	}
	// FIXME arch, owner, title
	s.Cache.InsertServer(server.Server.Identifier, server.Server.Location.ZoneID, server.Server.Arch, server.Server.Organization, server.Server.Name)
	return server.Server.Identifier, nil
}

// PatchUserSSHKey updates a user
func (s *ScalewayAPI) PatchUserSSHKey(UserID string, definition types.ScalewayUserPatchSSHKeyDefinition) error {
	resp, err := s.PatchResponse(AccountAPI, fmt.Sprintf("users/%s", UserID), definition)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusOK}, resp); err != nil {
		return err
	}
	return nil
}

// PatchServer updates a server
func (s *ScalewayAPI) PatchServer(serverID string, definition types.ScalewayServerPatchDefinition) error {
	resp, err := s.PatchResponse(s.computeAPI, fmt.Sprintf("servers/%s", serverID), definition)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusOK}, resp); err != nil {
		return err
	}
	return nil
}

// PostSnapshot creates a new snapshot
func (s *ScalewayAPI) PostSnapshot(volumeID string, name string) (string, error) {
	definition := types.ScalewaySnapshotDefinition{
		VolumeIDentifier: volumeID,
		Name:             name,
		Organization:     s.Organization,
	}
	resp, err := s.PostResponse(s.computeAPI, "snapshots", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var snapshot types.ScalewayOneSnapshot

	if err = json.Unmarshal(body, &snapshot); err != nil {
		return "", err
	}
	// FIXME arch, owner, title
	s.Cache.InsertSnapshot(snapshot.Snapshot.Identifier, "", "", snapshot.Snapshot.Organization, snapshot.Snapshot.Name)
	return snapshot.Snapshot.Identifier, nil
}

// PostImage creates a new image
func (s *ScalewayAPI) PostImage(volumeID string, name string, bootscript string, arch string) (string, error) {
	definition := types.ScalewayImageDefinition{
		SnapshotIDentifier: volumeID,
		Name:               name,
		Organization:       s.Organization,
		Arch:               arch,
	}
	if bootscript != "" {
		definition.DefaultBootscript = &bootscript
	}

	resp, err := s.PostResponse(s.computeAPI, "images", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var image types.ScalewayOneImage

	if err = json.Unmarshal(body, &image); err != nil {
		return "", err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertImage(image.Image.Identifier, "", image.Image.Arch, image.Image.Organization, image.Image.Name, "")
	return image.Image.Identifier, nil
}

// PostVolume creates a new volume
func (s *ScalewayAPI) PostVolume(definition types.ScalewayVolumeDefinition) (string, error) {
	definition.Organization = s.Organization
	if definition.Type == "" {
		definition.Type = "l_ssd"
	}

	resp, err := s.PostResponse(s.computeAPI, "volumes", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var volume types.ScalewayOneVolume

	if err = json.Unmarshal(body, &volume); err != nil {
		return "", err
	}
	// FIXME: s.Cache.InsertVolume(volume.Volume.Identifier, volume.Volume.Name)
	return volume.Volume.Identifier, nil
}

// PutVolume updates a volume
func (s *ScalewayAPI) PutVolume(volumeID string, definition types.ScalewayVolumePutDefinition) error {
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("volumes/%s", volumeID), definition)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// ResolveServer attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveServer(needle string) (types.ScalewayResolverResults, error) {
	servers, err := s.Cache.LookUpServers(needle, true)
	if err != nil {
		return servers, err
	}
	if len(servers) == 0 {
		if _, err = s.GetServers(true, 0); err != nil {
			return nil, err
		}
		servers, err = s.Cache.LookUpServers(needle, true)
	}
	return servers, err
}

// ResolveVolume attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveVolume(needle string) (types.ScalewayResolverResults, error) {
	volumes, err := s.Cache.LookUpVolumes(needle, true)
	if err != nil {
		return volumes, err
	}
	if len(volumes) == 0 {
		if _, err = s.GetVolumes(); err != nil {
			return nil, err
		}
		volumes, err = s.Cache.LookUpVolumes(needle, true)
	}
	return volumes, err
}

// ResolveSnapshot attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveSnapshot(needle string) (types.ScalewayResolverResults, error) {
	snapshots, err := s.Cache.LookUpSnapshots(needle, true)
	if err != nil {
		return snapshots, err
	}
	if len(snapshots) == 0 {
		if _, err = s.GetSnapshots(); err != nil {
			return nil, err
		}
		snapshots, err = s.Cache.LookUpSnapshots(needle, true)
	}
	return snapshots, err
}

// ResolveImage attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveImage(needle string) (types.ScalewayResolverResults, error) {
	images, err := s.Cache.LookUpImages(needle, true)
	if err != nil {
		return images, err
	}
	if len(images) == 0 {
		if _, err = s.GetImages(); err != nil {
			return nil, err
		}
		images, err = s.Cache.LookUpImages(needle, true)
	}
	return images, err
}

// ResolveBootscript attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveBootscript(needle string) (types.ScalewayResolverResults, error) {
	bootscripts, err := s.Cache.LookUpBootscripts(needle, true)
	if err != nil {
		return bootscripts, err
	}
	if len(bootscripts) == 0 {
		if _, err = s.GetBootscripts(); err != nil {
			return nil, err
		}
		bootscripts, err = s.Cache.LookUpBootscripts(needle, true)
	}
	return bootscripts, err
}

// GetImages gets the list of images from the ScalewayAPI
func (s *ScalewayAPI) GetImages() (*[]types.MarketImage, error) {
	images, err := s.GetMarketPlaceImages("")
	if err != nil {
		return nil, err
	}
	s.Cache.ClearImages()
	for i, image := range images.Images {
		if image.CurrentPublicVersion != "" {
			for _, version := range image.Versions {
				if version.ID == image.CurrentPublicVersion {
					for _, localImage := range version.LocalImages {
						images.Images[i].Public = true
						s.Cache.InsertImage(localImage.ID, localImage.Zone, localImage.Arch, image.Organization.ID, image.Name, image.CurrentPublicVersion)
					}
				}
			}
		}
	}
	values := url.Values{}
	values.Set("organization", s.Organization)
	resp, err := s.GetResponsePaginate(s.computeAPI, "images", values)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var OrgaImages types.ScalewayImages

	if err = json.Unmarshal(body, &OrgaImages); err != nil {
		return nil, err
	}

	for _, orgaImage := range OrgaImages.Images {
		images.Images = append(images.Images, types.MarketImage{
			Categories:           []string{"MyImages"},
			CreationDate:         orgaImage.CreationDate,
			CurrentPublicVersion: orgaImage.Identifier,
			ModificationDate:     orgaImage.ModificationDate,
			Name:                 orgaImage.Name,
			Public:               false,
			MarketVersions: types.MarketVersions{
				Versions: []types.MarketVersionDefinition{
					{
						CreationDate:     orgaImage.CreationDate,
						ID:               orgaImage.Identifier,
						ModificationDate: orgaImage.ModificationDate,
						MarketLocalImages: types.MarketLocalImages{
							LocalImages: []types.MarketLocalImageDefinition{
								{
									Arch: orgaImage.Arch,
									ID:   orgaImage.Identifier,
									// TODO: fecth images from ams1 and par1
									Zone: s.Region,
								},
							},
						},
					},
				},
			},
		})
		s.Cache.InsertImage(orgaImage.Identifier, s.Region, orgaImage.Arch, orgaImage.Organization, orgaImage.Name, "")
	}
	return &images.Images, nil
}

// GetImage gets an image from the ScalewayAPI
func (s *ScalewayAPI) GetImage(imageID string) (*types.ScalewayImage, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "images/"+imageID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneImage types.ScalewayOneImage

	if err = json.Unmarshal(body, &oneImage); err != nil {
		return nil, err
	}
	// FIXME owner, title
	s.Cache.InsertImage(oneImage.Image.Identifier, s.Region, oneImage.Image.Arch, oneImage.Image.Organization, oneImage.Image.Name, "")
	return &oneImage.Image, nil
}

// DeleteImage deletes a image
func (s *ScalewayAPI) DeleteImage(imageID string) error {
	defer s.Cache.RemoveImage(imageID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("images/%s", imageID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// DeleteSnapshot deletes a snapshot
func (s *ScalewayAPI) DeleteSnapshot(snapshotID string) error {
	defer s.Cache.RemoveSnapshot(snapshotID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("snapshots/%s", snapshotID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// DeleteVolume deletes a volume
func (s *ScalewayAPI) DeleteVolume(volumeID string) error {
	defer s.Cache.RemoveVolume(volumeID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("volumes/%s", volumeID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// GetSnapshots gets the list of snapshots from the ScalewayAPI
func (s *ScalewayAPI) GetSnapshots() (*[]types.ScalewaySnapshot, error) {
	query := url.Values{}
	s.Cache.ClearSnapshots()

	resp, err := s.GetResponsePaginate(s.computeAPI, "snapshots", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var snapshots types.ScalewaySnapshots

	if err = json.Unmarshal(body, &snapshots); err != nil {
		return nil, err
	}
	for _, snapshot := range snapshots.Snapshots {
		// FIXME region, arch, owner, title
		s.Cache.InsertSnapshot(snapshot.Identifier, "", "", snapshot.Organization, snapshot.Name)
	}
	return &snapshots.Snapshots, nil
}

// GetSnapshot gets a snapshot from the ScalewayAPI
func (s *ScalewayAPI) GetSnapshot(snapshotID string) (*types.ScalewaySnapshot, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "snapshots/"+snapshotID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneSnapshot types.ScalewayOneSnapshot

	if err = json.Unmarshal(body, &oneSnapshot); err != nil {
		return nil, err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertSnapshot(oneSnapshot.Snapshot.Identifier, "", "", oneSnapshot.Snapshot.Organization, oneSnapshot.Snapshot.Name)
	return &oneSnapshot.Snapshot, nil
}

// GetVolumes gets the list of volumes from the ScalewayAPI
func (s *ScalewayAPI) GetVolumes() (*[]types.ScalewayVolume, error) {
	query := url.Values{}
	s.Cache.ClearVolumes()

	resp, err := s.GetResponsePaginate(s.computeAPI, "volumes", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}

	var volumes types.ScalewayVolumes

	if err = json.Unmarshal(body, &volumes); err != nil {
		return nil, err
	}
	for _, volume := range volumes.Volumes {
		// FIXME region, arch, owner, title
		s.Cache.InsertVolume(volume.Identifier, "", "", volume.Organization, volume.Name)
	}
	return &volumes.Volumes, nil
}

// GetVolume gets a volume from the ScalewayAPI
func (s *ScalewayAPI) GetVolume(volumeID string) (*types.ScalewayVolume, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "volumes/"+volumeID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneVolume types.ScalewayOneVolume

	if err = json.Unmarshal(body, &oneVolume); err != nil {
		return nil, err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertVolume(oneVolume.Volume.Identifier, "", "", oneVolume.Volume.Organization, oneVolume.Volume.Name)
	return &oneVolume.Volume, nil
}

// GetBootscripts gets the list of bootscripts from the ScalewayAPI
func (s *ScalewayAPI) GetBootscripts() (*[]types.ScalewayBootscript, error) {
	query := url.Values{}

	s.Cache.ClearBootscripts()
	resp, err := s.GetResponsePaginate(s.computeAPI, "bootscripts", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var bootscripts types.ScalewayBootscripts

	if err = json.Unmarshal(body, &bootscripts); err != nil {
		return nil, err
	}
	for _, bootscript := range bootscripts.Bootscripts {
		// FIXME region, arch, owner, title
		s.Cache.InsertBootscript(bootscript.Identifier, "", bootscript.Arch, bootscript.Organization, bootscript.Title)
	}
	return &bootscripts.Bootscripts, nil
}

// GetBootscript gets a bootscript from the ScalewayAPI
func (s *ScalewayAPI) GetBootscript(bootscriptID string) (*types.ScalewayBootscript, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "bootscripts/"+bootscriptID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneBootscript types.ScalewayOneBootscript

	if err = json.Unmarshal(body, &oneBootscript); err != nil {
		return nil, err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertBootscript(oneBootscript.Bootscript.Identifier, "", oneBootscript.Bootscript.Arch, oneBootscript.Bootscript.Organization, oneBootscript.Bootscript.Title)
	return &oneBootscript.Bootscript, nil
}

// GetUserdatas gets list of userdata for a server
func (s *ScalewayAPI) GetUserdatas(serverID string, metadata bool) (*types.ScalewayUserdatas, error) {
	var uri, endpoint string

	endpoint = s.computeAPI
	if metadata {
		uri = "/user_data"
		endpoint = MetadataAPI
	} else {
		uri = fmt.Sprintf("servers/%s/user_data", serverID)
	}

	resp, err := s.GetResponsePaginate(endpoint, uri, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var userdatas types.ScalewayUserdatas

	if err = json.Unmarshal(body, &userdatas); err != nil {
		return nil, err
	}
	return &userdatas, nil
}

// GetUserdata gets a specific userdata for a server
func (s *ScalewayAPI) GetUserdata(serverID, key string, metadata bool) (*types.ScalewayUserdata, error) {
	var uri, endpoint string

	endpoint = s.computeAPI
	if metadata {
		uri = fmt.Sprintf("/user_data/%s", key)
		endpoint = MetadataAPI
	} else {
		uri = fmt.Sprintf("servers/%s/user_data/%s", serverID, key)
	}

	var err error
	resp, err := s.GetResponsePaginate(endpoint, uri, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("no such user_data %q (%d)", key, resp.StatusCode)
	}
	var data types.ScalewayUserdata
	data, err = ioutil.ReadAll(resp.Body)
	return &data, err
}

// PatchUserdata sets a user data
func (s *ScalewayAPI) PatchUserdata(serverID, key string, value []byte, metadata bool) error {
	var resource, endpoint string

	endpoint = s.computeAPI
	if metadata {
		resource = fmt.Sprintf("/user_data/%s", key)
		endpoint = MetadataAPI
	} else {
		resource = fmt.Sprintf("servers/%s/user_data/%s", serverID, key)
	}

	uri := fmt.Sprintf("%s/%s", strings.TrimRight(endpoint, "/"), resource)
	payload := new(bytes.Buffer)
	payload.Write(value)

	req, err := http.NewRequest("PATCH", uri, payload)
	if err != nil {
		return err
	}

	req.Header.Set("X-Auth-Token", s.Token)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", s.userAgent)

	s.LogHTTP(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return fmt.Errorf("cannot set user_data (%d)", resp.StatusCode)
}

// DeleteUserdata deletes a server user_data
func (s *ScalewayAPI) DeleteUserdata(serverID, key string, metadata bool) error {
	var url, endpoint string

	endpoint = s.computeAPI
	if metadata {
		url = fmt.Sprintf("/user_data/%s", key)
		endpoint = MetadataAPI
	} else {
		url = fmt.Sprintf("servers/%s/user_data/%s", serverID, key)
	}

	resp, err := s.DeleteResponse(endpoint, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// GetTasks get the list of tasks from the ScalewayAPI
func (s *ScalewayAPI) GetTasks() (*[]types.ScalewayTask, error) {
	query := url.Values{}
	resp, err := s.GetResponsePaginate(s.computeAPI, "tasks", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var tasks types.ScalewayTasks

	if err = json.Unmarshal(body, &tasks); err != nil {
		return nil, err
	}
	return &tasks.Tasks, nil
}

// CheckCredentials performs a dummy check to ensure we can contact the API
func (s *ScalewayAPI) CheckCredentials() error {
	query := url.Values{}

	resp, err := s.GetResponsePaginate(AccountAPI, "tokens", query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return err
	}
	found := false
	var tokens types.ScalewayGetTokens

	if err = json.Unmarshal(body, &tokens); err != nil {
		return err
	}
	for _, token := range tokens.Tokens {
		if token.ID == s.Token {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Invalid token %v", s.Token)
	}
	return nil
}

// GetUserID returns the userID
func (s *ScalewayAPI) GetUserID() (string, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("tokens/%s", s.Token), url.Values{})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return "", err
	}
	var token types.ScalewayTokensDefinition

	if err = json.Unmarshal(body, &token); err != nil {
		return "", err
	}
	return token.Token.UserID, nil
}

// GetOrganization returns Organization
func (s *ScalewayAPI) GetOrganization() (*types.ScalewayOrganizationsDefinition, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, "organizations", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var data types.ScalewayOrganizationsDefinition

	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetUser returns the user
func (s *ScalewayAPI) GetUser() (*types.ScalewayUserDefinition, error) {
	userID, err := s.GetUserID()
	if err != nil {
		return nil, err
	}
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("users/%s", userID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var user types.ScalewayUsersDefinition

	if err = json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return &user.User, nil
}

// GetPermissions returns the permissions
func (s *ScalewayAPI) GetPermissions() (*types.ScalewayPermissionDefinition, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("tokens/%s/permissions", s.Token), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var permissions types.ScalewayPermissionDefinition

	if err = json.Unmarshal(body, &permissions); err != nil {
		return nil, err
	}
	return &permissions, nil
}

// GetDashboard returns the dashboard
func (s *ScalewayAPI) GetDashboard() (*types.ScalewayDashboard, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "dashboard", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var dashboard types.ScalewayDashboardResp

	if err = json.Unmarshal(body, &dashboard); err != nil {
		return nil, err
	}
	return &dashboard.Dashboard, nil
}

// GetServerID returns exactly one server matching
func (s *ScalewayAPI) GetServerID(needle string) (string, error) {
	// Parses optional type prefix, i.e: "server:name" -> "name"
	_, needle = types.ParseNeedle(needle)

	servers, err := s.ResolveServer(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve server %s: %s", needle, err)
	}
	if len(servers) == 1 {
		return servers[0].Identifier, nil
	}
	if len(servers) == 0 {
		return "", fmt.Errorf("No such server: %s", needle)
	}
	return "", showResolverResults(needle, servers)
}

func showResolverResults(needle string, results types.ScalewayResolverResults) error {
	w := tabwriter.NewWriter(os.Stderr, 20, 1, 3, ' ', 0)
	defer w.Flush()
	sort.Sort(results)
	fmt.Fprintf(w, "  IMAGEID\tFROM\tNAME\tZONE\tARCH\n")
	for _, result := range results {
		if result.Arch == "" {
			result.Arch = "n/a"
		}
		fmt.Fprintf(w, "- %s\t%s\t%s\t%s\t%s\n", result.TruncIdentifier(), result.CodeName(), result.Name, result.Region, result.Arch)
	}
	return fmt.Errorf("Too many candidates for %s (%d)", needle, len(results))
}

// GetVolumeID returns exactly one volume matching
func (s *ScalewayAPI) GetVolumeID(needle string) (string, error) {
	// Parses optional type prefix, i.e: "volume:name" -> "name"
	_, needle = types.ParseNeedle(needle)

	volumes, err := s.ResolveVolume(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve volume %s: %s", needle, err)
	}
	if len(volumes) == 1 {
		return volumes[0].Identifier, nil
	}
	if len(volumes) == 0 {
		return "", fmt.Errorf("No such volume: %s", needle)
	}
	return "", showResolverResults(needle, volumes)
}

// GetSnapshotID returns exactly one snapshot matching
func (s *ScalewayAPI) GetSnapshotID(needle string) (string, error) {
	// Parses optional type prefix, i.e: "snapshot:name" -> "name"
	_, needle = types.ParseNeedle(needle)

	snapshots, err := s.ResolveSnapshot(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve snapshot %s: %s", needle, err)
	}
	if len(snapshots) == 1 {
		return snapshots[0].Identifier, nil
	}
	if len(snapshots) == 0 {
		return "", fmt.Errorf("No such snapshot: %s", needle)
	}
	return "", showResolverResults(needle, snapshots)
}

// FilterImagesByArch removes entry that doesn't match with architecture
func FilterImagesByArch(res types.ScalewayResolverResults, arch string) (ret types.ScalewayResolverResults) {
	if arch == "*" {
		return res
	}
	for _, result := range res {
		if result.Arch == arch {
			ret = append(ret, result)
		}
	}
	return
}

// FilterImagesByRegion removes entry that doesn't match with region
func FilterImagesByRegion(res types.ScalewayResolverResults, region string) (ret types.ScalewayResolverResults) {
	if region == "*" {
		return res
	}
	for _, result := range res {
		if result.Region == region {
			ret = append(ret, result)
		}
	}
	return
}

// GetImageID returns exactly one image matching
func (s *ScalewayAPI) GetImageID(needle, arch string) (*types.ScalewayImageIdentifier, error) {
	// Parses optional type prefix, i.e: "image:name" -> "name"
	_, needle = types.ParseNeedle(needle)

	images, err := s.ResolveImage(needle)
	if err != nil {
		return nil, fmt.Errorf("Unable to resolve image %s: %s", needle, err)
	}
	images = FilterImagesByArch(images, arch)
	images = FilterImagesByRegion(images, s.Region)
	if len(images) == 1 {
		return &types.ScalewayImageIdentifier{
			Identifier: images[0].Identifier,
			Arch:       images[0].Arch,
			// FIXME region, owner hardcoded
			Region: images[0].Region,
			Owner:  "",
		}, nil
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("No such image (zone %s, arch %s) : %s", s.Region, arch, needle)
	}
	return nil, showResolverResults(needle, images)
}

// GetSecurityGroups returns a ScalewaySecurityGroups
func (s *ScalewayAPI) GetSecurityGroups() (*types.ScalewayGetSecurityGroups, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "security_groups", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroups types.ScalewayGetSecurityGroups

	if err = json.Unmarshal(body, &securityGroups); err != nil {
		return nil, err
	}
	return &securityGroups, nil
}

// GetSecurityGroupRules returns a ScalewaySecurityGroupRules
func (s *ScalewayAPI) GetSecurityGroupRules(groupID string) (*types.ScalewayGetSecurityGroupRules, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("security_groups/%s/rules", groupID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroupRules types.ScalewayGetSecurityGroupRules

	if err = json.Unmarshal(body, &securityGroupRules); err != nil {
		return nil, err
	}
	return &securityGroupRules, nil
}

// GetASecurityGroupRule returns a ScalewaySecurityGroupRule
func (s *ScalewayAPI) GetASecurityGroupRule(groupID string, rulesID string) (*types.ScalewayGetSecurityGroupRule, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("security_groups/%s/rules/%s", groupID, rulesID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroupRules types.ScalewayGetSecurityGroupRule

	if err = json.Unmarshal(body, &securityGroupRules); err != nil {
		return nil, err
	}
	return &securityGroupRules, nil
}

// GetASecurityGroup returns a ScalewaySecurityGroup
func (s *ScalewayAPI) GetASecurityGroup(groupsID string) (*types.ScalewayGetSecurityGroup, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("security_groups/%s", groupsID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroups types.ScalewayGetSecurityGroup

	if err = json.Unmarshal(body, &securityGroups); err != nil {
		return nil, err
	}
	return &securityGroups, nil
}

// PostSecurityGroup posts a group on a server
func (s *ScalewayAPI) PostSecurityGroup(group types.ScalewayNewSecurityGroup) error {
	resp, err := s.PostResponse(s.computeAPI, "security_groups", group)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusCreated}, resp)
	return err
}

// PostSecurityGroupRule posts a rule on a server
func (s *ScalewayAPI) PostSecurityGroupRule(SecurityGroupID string, rules types.ScalewayNewSecurityGroupRule) error {
	resp, err := s.PostResponse(s.computeAPI, fmt.Sprintf("security_groups/%s/rules", SecurityGroupID), rules)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusCreated}, resp)
	return err
}

// DeleteSecurityGroup deletes a SecurityGroup
func (s *ScalewayAPI) DeleteSecurityGroup(securityGroupID string) error {
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("security_groups/%s", securityGroupID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// PutSecurityGroup updates a SecurityGroup
func (s *ScalewayAPI) PutSecurityGroup(group types.ScalewayUpdateSecurityGroup, securityGroupID string) error {
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("security_groups/%s", securityGroupID), group)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// PutSecurityGroupRule updates a SecurityGroupRule
func (s *ScalewayAPI) PutSecurityGroupRule(rules types.ScalewayNewSecurityGroupRule, securityGroupID, RuleID string) error {
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("security_groups/%s/rules/%s", securityGroupID, RuleID), rules)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DeleteSecurityGroupRule deletes a SecurityGroupRule
func (s *ScalewayAPI) DeleteSecurityGroupRule(SecurityGroupID, RuleID string) error {
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("security_groups/%s/rules/%s", SecurityGroupID, RuleID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// GetContainers returns a types.ScalewayGetContainers
func (s *ScalewayAPI) GetContainers() (*types.ScalewayGetContainers, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "containers", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var containers types.ScalewayGetContainers

	if err = json.Unmarshal(body, &containers); err != nil {
		return nil, err
	}
	return &containers, nil
}

// GetContainerDatas returns a types.ScalewayGetContainerDatas
func (s *ScalewayAPI) GetContainerDatas(container string) (*types.ScalewayGetContainerDatas, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("containers/%s", container), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var datas types.ScalewayGetContainerDatas

	if err = json.Unmarshal(body, &datas); err != nil {
		return nil, err
	}
	return &datas, nil
}

// GetIPS returns a types.ScalewayGetIPS
func (s *ScalewayAPI) GetIPS() (*types.ScalewayGetIPS, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "ips", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ips types.ScalewayGetIPS

	if err = json.Unmarshal(body, &ips); err != nil {
		return nil, err
	}
	return &ips, nil
}

// NewIP returns a new IP
func (s *ScalewayAPI) NewIP() (*types.ScalewayGetIP, error) {
	var orga struct {
		Organization string `json:"organization"`
	}
	orga.Organization = s.Organization
	resp, err := s.PostResponse(s.computeAPI, "ips", orga)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return nil, err
	}
	var ip types.ScalewayGetIP

	if err = json.Unmarshal(body, &ip); err != nil {
		return nil, err
	}
	return &ip, nil
}

// AttachIP attachs an IP to a server
func (s *ScalewayAPI) AttachIP(ipID, serverID string) error {
	var update struct {
		Address      string  `json:"address"`
		ID           string  `json:"id"`
		Reverse      *string `json:"reverse"`
		Organization string  `json:"organization"`
		Server       string  `json:"server"`
	}

	ip, err := s.GetIP(ipID)
	if err != nil {
		return err
	}
	update.Address = ip.IP.Address
	update.ID = ip.IP.ID
	update.Organization = ip.IP.Organization
	update.Server = serverID
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("ips/%s", ipID), update)
	if err != nil {
		return err
	}
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DetachIP detaches an IP from a server
func (s *ScalewayAPI) DetachIP(ipID string) error {
	ip, err := s.GetIP(ipID)
	if err != nil {
		return err
	}
	ip.IP.Server = nil
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("ips/%s", ipID), ip.IP)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DeleteIP deletes an IP
func (s *ScalewayAPI) DeleteIP(ipID string) error {
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("ips/%s", ipID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// GetIP returns a types.ScalewayGetIP
func (s *ScalewayAPI) GetIP(ipID string) (*types.ScalewayGetIP, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("ips/%s", ipID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ip types.ScalewayGetIP

	if err = json.Unmarshal(body, &ip); err != nil {
		return nil, err
	}
	return &ip, nil
}

// GetQuotas returns a types.ScalewayGetQuotas
func (s *ScalewayAPI) GetQuotas() (*types.ScalewayGetQuotas, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("organizations/%s/quotas", s.Organization), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var quotas types.ScalewayGetQuotas

	if err = json.Unmarshal(body, &quotas); err != nil {
		return nil, err
	}
	return &quotas, nil
}

// GetBootscriptID returns exactly one bootscript matching
func (s *ScalewayAPI) GetBootscriptID(needle, arch string) (string, error) {
	// Parses optional type prefix, i.e: "bootscript:name" -> "name"
	_, needle = types.ParseNeedle(needle)

	bootscripts, err := s.ResolveBootscript(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve bootscript %s: %s", needle, err)
	}
	bootscripts.FilterByArch(arch)
	if len(bootscripts) == 1 {
		return bootscripts[0].Identifier, nil
	}
	if len(bootscripts) == 0 {
		return "", fmt.Errorf("No such bootscript: %s", needle)
	}
	return "", showResolverResults(needle, bootscripts)
}

func rootNetDial(network, addr string) (net.Conn, error) {
	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
	}

	// bruteforce privileged ports
	var localAddr net.Addr
	var err error
	for port := 1; port <= 1024; port++ {
		localAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))

		// this should never happen
		if err != nil {
			return nil, err
		}

		dialer.LocalAddr = localAddr

		conn, err := dialer.Dial(network, addr)

		// if err is nil, dialer.Dial succeed, so let's go
		// else, err != nil, but we don't care
		if err == nil {
			return conn, nil
		}
	}
	// if here, all privileged ports were tried without success
	return nil, fmt.Errorf("bind: permission denied, are you root ?")
}

// SetPassword register the password
func (s *ScalewayAPI) SetPassword(password string) {
	s.password = password
}

// GetMarketPlaceImages returns images from marketplace
func (s *ScalewayAPI) GetMarketPlaceImages(uuidImage string) (*types.MarketImages, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%s", uuidImage), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret types.MarketImages

	if uuidImage != "" {
		ret.Images = make([]types.MarketImage, 1)

		var img types.MarketImage

		if err = json.Unmarshal(body, &img); err != nil {
			return nil, err
		}
		ret.Images[0] = img
	} else {
		if err = json.Unmarshal(body, &ret); err != nil {
			return nil, err
		}
	}
	return &ret, nil
}

// GetMarketPlaceImageVersions returns image version
func (s *ScalewayAPI) GetMarketPlaceImageVersions(uuidImage, uuidVersion string) (*types.MarketVersions, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s", uuidImage, uuidVersion), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret types.MarketVersions

	if uuidImage != "" {
		var version types.MarketVersion
		ret.Versions = make([]types.MarketVersionDefinition, 1)

		if err = json.Unmarshal(body, &version); err != nil {
			return nil, err
		}
		ret.Versions[0] = version.Version
	} else {
		if err = json.Unmarshal(body, &ret); err != nil {
			return nil, err
		}
	}
	return &ret, nil
}

// GetMarketPlaceImageCurrentVersion return the image current version
func (s *ScalewayAPI) GetMarketPlaceImageCurrentVersion(uuidImage string) (*types.MarketVersion, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%v/versions/current", uuidImage), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret types.MarketVersion

	if err = json.Unmarshal(body, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

// GetMarketPlaceLocalImages returns images from local region
func (s *ScalewayAPI) GetMarketPlaceLocalImages(uuidImage, uuidVersion, uuidLocalImage string) (*types.MarketLocalImages, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%s", uuidImage, uuidVersion, uuidLocalImage), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret types.MarketLocalImages
	if uuidLocalImage != "" {
		var localImage types.MarketLocalImage
		ret.LocalImages = make([]types.MarketLocalImageDefinition, 1)

		if err = json.Unmarshal(body, &localImage); err != nil {
			return nil, err
		}
		ret.LocalImages[0] = localImage.LocalImages
	} else {
		if err = json.Unmarshal(body, &ret); err != nil {
			return nil, err
		}
	}
	return &ret, nil
}

// PostMarketPlaceImage adds new image
func (s *ScalewayAPI) PostMarketPlaceImage(images types.MarketImage) error {
	resp, err := s.PostResponse(MarketplaceAPI, "images/", images)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// PostMarketPlaceImageVersion adds new image version
func (s *ScalewayAPI) PostMarketPlaceImageVersion(uuidImage string, version types.MarketVersion) error {
	resp, err := s.PostResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions", uuidImage), version)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// PostMarketPlaceLocalImage adds new local image
func (s *ScalewayAPI) PostMarketPlaceLocalImage(uuidImage, uuidVersion, uuidLocalImage string, local types.MarketLocalImage) error {
	resp, err := s.PostResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%v", uuidImage, uuidVersion, uuidLocalImage), local)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// PutMarketPlaceImage updates image
func (s *ScalewayAPI) PutMarketPlaceImage(uudiImage string, images types.MarketImage) error {
	resp, err := s.PutResponse(MarketplaceAPI, fmt.Sprintf("images/%v", uudiImage), images)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// PutMarketPlaceImageVersion updates image version
func (s *ScalewayAPI) PutMarketPlaceImageVersion(uuidImage, uuidVersion string, version types.MarketVersion) error {
	resp, err := s.PutResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%v", uuidImage, uuidVersion), version)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// PutMarketPlaceLocalImage updates local image
func (s *ScalewayAPI) PutMarketPlaceLocalImage(uuidImage, uuidVersion, uuidLocalImage string, local types.MarketLocalImage) error {
	resp, err := s.PostResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%v", uuidImage, uuidVersion, uuidLocalImage), local)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DeleteMarketPlaceImage deletes image
func (s *ScalewayAPI) DeleteMarketPlaceImage(uudImage string) error {
	resp, err := s.DeleteResponse(MarketplaceAPI, fmt.Sprintf("images/%v", uudImage))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// DeleteMarketPlaceImageVersion delete image version
func (s *ScalewayAPI) DeleteMarketPlaceImageVersion(uuidImage, uuidVersion string) error {
	resp, err := s.DeleteResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%v", uuidImage, uuidVersion))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// DeleteMarketPlaceLocalImage deletes local image
func (s *ScalewayAPI) DeleteMarketPlaceLocalImage(uuidImage, uuidVersion, uuidLocalImage string) error {
	resp, err := s.DeleteResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%v", uuidImage, uuidVersion, uuidLocalImage))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// ResolveTTYUrl return an URL to get a tty
func (s *ScalewayAPI) ResolveTTYUrl() string {
	switch s.Region {
	case "par1", "":
		return "https://tty-par1.scaleway.com/v2/"
	case "ams1":
		return "https://tty-ams1.scaleway.com"
	}
	return ""
}

// GetProductServers Fetches all the server type and their constraints from the Products API
func (s *ScalewayAPI) GetProductsServers() (*types.ScalewayProductsServers, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "products/servers", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}

	var productServers types.ScalewayProductsServers
	if err = json.Unmarshal(body, &productServers); err != nil {
		return nil, err
	}

	return &productServers, nil
}

// HideAPICredentials removes API credentials from a string
func (s *ScalewayAPI) HideAPICredentials(input string) string {
	output := input
	if s.Token != "" {
		output = strings.Replace(output, s.Token, "00000000-0000-4000-8000-000000000000", -1)
	}
	if s.Organization != "" {
		output = strings.Replace(output, s.Organization, "00000000-0000-5000-9000-000000000000", -1)
	}
	if s.password != "" {
		output = strings.Replace(output, s.password, "XX-XX-XX-XX", -1)
	}
	return output
}
