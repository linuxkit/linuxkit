package docker

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/dockerversion"
	"github.com/docker/docker/registry"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"

	"github.com/estesp/manifest-tool/types"
)

// we will store up a list of blobs we must ask the registry
// to cross-mount into our target namespace
type blobMount struct {
	FromRepo string
	Digest   string
}

// if we have mounted blobs referenced from manifests from
// outside the target repository namespace we will need to
// push them to our target's repo as they will be references
// from the final manifest list object we push
type manifestPush struct {
	Name      string
	Digest    string
	JSONBytes []byte
	MediaType string
}

// PutManifestList takes an authentication variable and a yaml spec struct and pushes an image list based on the spec
func PutManifestList(a *types.AuthInfo, yamlInput types.YAMLInput, ignoreMissing, insecure bool) (string, int, error) {
	var (
		manifestList      manifestlist.ManifestList
		blobMountRequests []blobMount
		manifestRequests  []manifestPush
	)

	// process the final image name reference for the manifest list
	targetRef, err := reference.ParseNormalizedNamed(yamlInput.Image)
	if err != nil {
		return "", 0, fmt.Errorf("Error parsing name for manifest list (%s): %v", yamlInput.Image, err)
	}
	targetRepo, err := registry.ParseRepositoryInfo(targetRef)
	if err != nil {
		return "", 0, fmt.Errorf("Error parsing repository name for manifest list (%s): %v", yamlInput.Image, err)
	}
	targetEndpoint, repoName, err := setupRepo(targetRepo, insecure)
	if err != nil {
		return "", 0, fmt.Errorf("Error setting up repository endpoint and references for %q: %v", targetRef, err)
	}

	// Now create the manifest list payload by looking up the manifest schemas
	// for the constituent images:
	logrus.Info("Retrieving digests of images...")
	for _, img := range yamlInput.Manifests {
		mfstData, repoInfo, err := GetImageData(a, img.Image, insecure, false)
		if err != nil {
			// if ignoreMissing is true, we will skip this error and simply
			// log a warning that we couldn't find it in the registry
			if ignoreMissing {
				logrus.Warnf("Couldn't find or access image reference %q. Skipping image.", img.Image)
				continue
			}
			return "", 0, fmt.Errorf("Inspect of image %q failed with error: %v", img.Image, err)
		}
		if reference.Domain(repoInfo.Name) != reference.Domain(targetRepo.Name) {
			return "", 0, fmt.Errorf("Cannot use source images from a different registry than the target image: %s != %s", reference.Domain(repoInfo.Name), reference.Domain(targetRepo.Name))
		}
		if len(mfstData) > 1 {
			// too many responses--can only happen if a manifest list was returned for the name lookup
			return "", 0, fmt.Errorf("You specified a manifest list entry from a digest that points to a current manifest list. Manifest lists do not allow recursion")
		}
		// the non-manifest list case will always have exactly one manifest response
		imgMfst := mfstData[0]

		// fill os/arch from inspected image if not specified in input YAML
		if img.Platform.OS == "" && img.Platform.Architecture == "" {
			// prefer a full platform object, if one is already available (and appears to have meaningful content)
			if imgMfst.Platform.OS != "" || imgMfst.Platform.Architecture != "" {
				img.Platform = imgMfst.Platform
			} else if imgMfst.Os != "" || imgMfst.Architecture != "" {
				img.Platform.OS = imgMfst.Os
				img.Platform.Architecture = imgMfst.Architecture
			}
		}

		// if the origin image has OSFeature and/or OSVersion information, and
		// these values were not specified in the creation YAML, then
		// retain the origin values in the Platform definition for the manifest list:
		if imgMfst.OSVersion != "" && img.Platform.OSVersion == "" {
			img.Platform.OSVersion = imgMfst.OSVersion
		}
		if len(imgMfst.OSFeatures) > 0 && len(img.Platform.OSFeatures) == 0 {
			img.Platform.OSFeatures = imgMfst.OSFeatures
		}

		// validate os/arch input
		if !isValidOSArch(img.Platform.OS, img.Platform.Architecture, img.Platform.Variant) {
			return "", 0, fmt.Errorf("Manifest entry for image %s has unsupported os/arch or os/arch/variant combination: %s/%s/%s", img.Image, img.Platform.OS, img.Platform.Architecture, img.Platform.Variant)
		}

		manifest := manifestlist.ManifestDescriptor{
			Platform: img.Platform,
		}
		manifest.Descriptor.Digest, err = digest.Parse(imgMfst.Digest)
		manifest.Size = imgMfst.Size
		manifest.MediaType = imgMfst.MediaType

		if err != nil {
			return "", 0, fmt.Errorf("Digest parse of image %q failed with error: %v", img.Image, err)
		}
		logrus.Infof("Image %q is digest %s; size: %d", img.Image, imgMfst.Digest, imgMfst.Size)

		// if this image is in a different repo, we need to add the layer & config digests to the list of
		// requested blob mounts (cross-repository push) before pushing the manifest list
		if repoName != reference.Path(repoInfo.Name) {
			logrus.Debugf("Adding manifest references of %q to blob mount requests", img.Image)
			for _, layer := range imgMfst.References {
				blobMountRequests = append(blobMountRequests, blobMount{FromRepo: reference.Path(repoInfo.Name), Digest: layer})
			}
			// also must add the manifest to be pushed in the target namespace
			logrus.Debugf("Adding manifest %q -> to be pushed to %q as a manifest reference", reference.Path(repoInfo.Name), repoName)
			manifestRequests = append(manifestRequests, manifestPush{
				Name:      reference.Path(repoInfo.Name),
				Digest:    imgMfst.Digest,
				JSONBytes: imgMfst.CanonicalJSON,
				MediaType: imgMfst.MediaType,
			})
		}
		manifestList.Manifests = append(manifestList.Manifests, manifest)
	}

	if ignoreMissing && len(manifestList.Manifests) == 0 {
		// we need to verify we at least have one valid entry in the list
		// otherwise our manifest list will be totally empty
		return "", 0, fmt.Errorf("all entries were skipped due to missing source image references; no manifest list to push")
	}
	// Set the schema version
	manifestList.Versioned = manifestlist.SchemaVersion

	urlBuilder, err := v2.NewURLBuilderFromString(targetEndpoint.URL.String(), false)
	if err != nil {
		return "", 0, fmt.Errorf("Can't create URL builder from endpoint (%s): %v", targetEndpoint.URL.String(), err)
	}
	pushURL, err := createManifestURLFromRef(targetRef, urlBuilder)
	if err != nil {
		return "", 0, fmt.Errorf("Error setting up repository endpoint and references for %q: %v", targetRef, err)
	}
	logrus.Debugf("Manifest list push url: %s", pushURL)

	deserializedManifestList, err := manifestlist.FromDescriptors(manifestList.Manifests)
	if err != nil {
		return "", 0, fmt.Errorf("Cannot deserialize manifest list: %v", err)
	}
	mediaType, p, err := deserializedManifestList.Payload()
	logrus.Debugf("mediaType of manifestList: %s", mediaType)
	if err != nil {
		return "", 0, fmt.Errorf("Cannot retrieve payload for HTTP PUT of manifest list: %v", err)

	}
	manifestLen := len(p)
	putRequest, err := http.NewRequest("PUT", pushURL, bytes.NewReader(p))
	if err != nil {
		return "", 0, fmt.Errorf("HTTP PUT request creation failed: %v", err)
	}
	putRequest.Header.Set("Content-Type", mediaType)

	httpClient, err := getHTTPClient(a, targetRepo, targetEndpoint, repoName)
	if err != nil {
		return "", 0, fmt.Errorf("Failed to setup HTTP client to repository: %v", err)
	}

	// before we push the manifest list, if we have any blob mount requests, we need
	// to ask the registry to mount those blobs in our target so they are available
	// as references
	if err := mountBlobs(httpClient, urlBuilder, targetRef, blobMountRequests); err != nil {
		return "", 0, fmt.Errorf("Couldn't mount blobs for cross-repository push: %v", err)
	}

	// we also must push any manifests that are referenced in the manifest list into
	// the target namespace
	if err := pushReferences(httpClient, urlBuilder, targetRef, manifestRequests); err != nil {
		return "", 0, fmt.Errorf("Couldn't push manifests referenced in our manifest list: %v", err)
	}

	resp, err := httpClient.Do(putRequest)
	if err != nil {
		return "", 0, fmt.Errorf("V2 registry PUT of manifest list failed: %v", err)
	}
	defer resp.Body.Close()

	var finalDigest string
	if statusSuccess(resp.StatusCode) {
		dgstHeader := resp.Header.Get("Docker-Content-Digest")
		dgst, err := digest.Parse(dgstHeader)
		if err != nil {
			return "", 0, err
		}
		finalDigest = string(dgst)
	} else {
		return "", 0, fmt.Errorf("Registry push unsuccessful: response %d: %s", resp.StatusCode, resp.Status)
	}
	// if the YAML includes additional tags, push the added tag references. No other work
	// should be required as we have already made sure all target blobs are cross-repo
	// mounted and all referenced manifests are already pushed.
	for _, tag := range yamlInput.Tags {
		newRef, err := reference.WithTag(targetRef, tag)
		if err != nil {
			return "", 0, fmt.Errorf("Error creating tagged reference for added tag %q: %v", tag, err)
		}
		pushURL, err := createManifestURLFromRef(newRef, urlBuilder)
		if err != nil {
			return "", 0, fmt.Errorf("Error setting up repository endpoint and references for %q: %v", newRef, err)
		}
		logrus.Debugf("[extra tag %q] push url: %s", tag, pushURL)
		putRequest, err := http.NewRequest("PUT", pushURL, bytes.NewReader(p))
		if err != nil {
			return "", 0, fmt.Errorf("[extra tag %q] HTTP PUT request creation failed: %v", tag, err)
		}
		putRequest.Header.Set("Content-Type", mediaType)
		resp, err := httpClient.Do(putRequest)
		if err != nil {
			return "", 0, fmt.Errorf("[extra tag %q] V2 registry PUT of manifest list failed: %v", tag, err)
		}
		defer resp.Body.Close()

		if statusSuccess(resp.StatusCode) {
			dgstHeader := resp.Header.Get("Docker-Content-Digest")
			dgst, err := digest.Parse(dgstHeader)
			if err != nil {
				return "", 0, err
			}
			if string(dgst) != finalDigest {
				logrus.Warnf("Extra tag %q push resulted in non-matching digest %s (should be %s", tag, string(dgst), finalDigest)
			}
		} else {
			return "", 0, fmt.Errorf("[extra tag %q] Registry push unsuccessful: response %d: %s", tag, resp.StatusCode, resp.Status)
		}
	}
	return finalDigest, manifestLen, nil
}

func getHTTPClient(a *types.AuthInfo, repoInfo *registry.RepositoryInfo, endpoint registry.APIEndpoint, repoName string) (*http.Client, error) {
	// get the http transport, this will be used in a client to upload manifest
	// TODO - add separate function get client
	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     endpoint.TLSConfig,
		DisableKeepAlives:   true,
	}
	authConfig, err := getAuthConfig(a, repoInfo.Index)
	if err != nil {
		return nil, fmt.Errorf("Cannot retrieve authconfig: %v", err)
	}
	modifiers := registry.Headers(dockerversion.DockerUserAgent(nil), http.Header{})
	authTransport := transport.NewTransport(base, modifiers...)
	challengeManager, _, err := registry.PingV2Registry(endpoint.URL, authTransport)
	if err != nil {
		return nil, fmt.Errorf("Ping of V2 registry failed: %v", err)
	}
	if authConfig.RegistryToken != "" {
		passThruTokenHandler := &existingTokenHandler{token: authConfig.RegistryToken}
		modifiers = append(modifiers, auth.NewAuthorizer(challengeManager, passThruTokenHandler))
	} else {
		creds := dumbCredentialStore{auth: &authConfig}
		tokenHandler := auth.NewTokenHandler(authTransport, creds, repoName, "push", "pull")
		basicHandler := auth.NewBasicHandler(creds)
		modifiers = append(modifiers, auth.NewAuthorizer(challengeManager, tokenHandler, basicHandler))
	}
	tr := transport.NewTransport(base, modifiers...)

	httpClient := &http.Client{
		Transport:     tr,
		CheckRedirect: checkHTTPRedirect,
	}
	return httpClient, nil
}

func createManifestURLFromRef(targetRef reference.Named, urlBuilder *v2.URLBuilder) (string, error) {
	// get rid of hostname so the target URL is constructed properly
	hostname, name := splitHostname(targetRef.String())
	targetRef, err := getNamedRefWithoutHostname(name)
	if err != nil {
		return "", fmt.Errorf("Can't parse target image repository name from reference: %v", err)
	}

	// Set the tag to latest, if no tag found in YAML
	if _, isTagged := targetRef.(reference.NamedTagged); !isTagged {
		targetRef = reference.TagNameOnly(targetRef)
	} else {
		tagged, _ := targetRef.(reference.NamedTagged)
		targetRef, err = reference.WithTag(targetRef, tagged.Tag())
		if err != nil {
			return "", fmt.Errorf("Error referencing specified tag to target repository name: %v", err)
		}
	}

	manifestURL, err := buildManifestURL(urlBuilder, hostname, targetRef)
	if err != nil {
		return "", fmt.Errorf("Failed to build manifest URL from target reference: %v", err)
	}
	return manifestURL, nil
}

func setupRepo(repoInfo *registry.RepositoryInfo, insecure bool) (registry.APIEndpoint, string, error) {

	options := registry.ServiceOptions{}
	if insecure {
		options.InsecureRegistries = append(options.InsecureRegistries, reference.Domain(repoInfo.Name))
	}
	registryService, err := registry.NewService(options)
	if err != nil {
		return registry.APIEndpoint{}, "", err
	}

	endpoints, err := registryService.LookupPushEndpoints(reference.Domain(repoInfo.Name))
	if err != nil {
		return registry.APIEndpoint{}, "", err
	}
	logrus.Debugf("endpoints: %v", endpoints)
	// take highest priority endpoint
	endpoint := endpoints[0]
	// if insecure, and there is an "http" endpoint, prefer that
	if insecure {
		for _, ep := range endpoints {
			if ep.URL.Scheme == "http" {
				endpoint = ep
			}
		}
		endpoint.TLSConfig.InsecureSkipVerify = true
	}

	repoName := repoInfo.Name.Name()
	// If endpoint does not support CanonicalName, use the Name's path instead
	if endpoint.TrimHostname {
		repoName = reference.Path(repoInfo.Name)
		logrus.Debugf("repoName: %v", repoName)
	}
	return endpoint, repoName, nil
}

func pushReferences(httpClient *http.Client, urlBuilder *v2.URLBuilder, ref reference.Named, manifests []manifestPush) error {
	// for each referenced manifest object in the manifest list (that is outside of our current repo/name)
	// we need to push by digest the manifest so that it is added as a valid reference in the current
	// repo. This will allow us to push the manifest list properly later and have all valid references.

	// first get rid of possible hostname so the target URL is constructed properly
	hostname, name := splitHostname(ref.String())
	ref, err := getNamedRefWithoutHostname(name)
	if err != nil {
		return fmt.Errorf("Error parsing repo/name portion of reference without hostname: %s: %v", name, err)
	}
	for _, manifest := range manifests {
		dgst, err := digest.Parse(manifest.Digest)
		if err != nil {
			return fmt.Errorf("Error parsing manifest digest (%s) for referenced manifest %q: %v", manifest.Digest, manifest.Name, err)
		}
		targetRef, err := reference.WithDigest(ref, dgst)
		if err != nil {
			return fmt.Errorf("Error creating manifest digest target for referenced manifest %q: %v", manifest.Name, err)
		}
		pushURL, err := buildManifestURL(urlBuilder, hostname, targetRef)
		if err != nil {
			return fmt.Errorf("Error setting up manifest push URL for manifest references for %q: %v", manifest.Name, err)
		}
		logrus.Debugf("manifest reference push URL: %s", pushURL)

		pushRequest, err := http.NewRequest("PUT", pushURL, bytes.NewReader(manifest.JSONBytes))
		if err != nil {
			return fmt.Errorf("HTTP PUT request creation for manifest reference push failed: %v", err)
		}
		pushRequest.Header.Set("Content-Type", manifest.MediaType)
		resp, err := httpClient.Do(pushRequest)
		if err != nil {
			return fmt.Errorf("PUT of manifest reference failed: %v", err)
		}

		resp.Body.Close()
		if !statusSuccess(resp.StatusCode) {
			return fmt.Errorf("Referenced manifest push unsuccessful: response %d: %s", resp.StatusCode, resp.Status)
		}
		dgstHeader := resp.Header.Get("Docker-Content-Digest")
		dgstResult, err := digest.Parse(dgstHeader)
		if err != nil {
			return fmt.Errorf("Couldn't parse pushed manifest digest response: %v", err)
		}
		if string(dgstResult) != manifest.Digest {
			return fmt.Errorf("Pushed referenced manifest received a different digest: expected %s, got %s", manifest.Digest, string(dgst))
		}
		logrus.Debugf("referenced manifest %q pushed; digest matches: %s", manifest.Name, string(dgst))
	}
	return nil
}

func mountBlobs(httpClient *http.Client, urlBuilder *v2.URLBuilder, ref reference.Named, blobsRequested []blobMount) error {
	// get rid of hostname so the target URL is constructed properly
	hostname, name := splitHostname(ref.String())
	targetRef, err := getNamedRefWithoutHostname(name)
	if err != nil {
		return fmt.Errorf("Can't parse reference without hostname: %v", err)
	}

	for _, blob := range blobsRequested {
		// create URL request
		url, err := buildBlobUploadURL(urlBuilder, hostname, targetRef, url.Values{"from": {blob.FromRepo}, "mount": {blob.Digest}})
		if err != nil {
			return fmt.Errorf("Failed to create blob mount URL: %v", err)
		}
		mountRequest, err := http.NewRequest("POST", url, nil)
		if err != nil {
			return fmt.Errorf("HTTP POST request creation for blob mount failed: %v", err)
		}
		mountRequest.Header.Set("Content-Length", "0")
		resp, err := httpClient.Do(mountRequest)
		if err != nil {
			return fmt.Errorf("V2 registry POST of blob mount failed: %v", err)
		}

		resp.Body.Close()
		if !statusSuccess(resp.StatusCode) {
			return fmt.Errorf("Blob mount failed to url %s: HTTP status %d", url, resp.StatusCode)
		}
		logrus.Debugf("Mount of blob %s succeeded, location: %q", blob.Digest, resp.Header.Get("Location"))
	}
	return nil
}

func buildManifestURL(ub *v2.URLBuilder, hostname string, targetRef reference.Named) (string, error) {
	if !isHubLibraryRef(targetRef, hostname) {
		return ub.BuildManifestURL(targetRef)
	}
	// this is a library reference and we don't want to lose the "library/" part of the URL ref
	baseURL, err := ub.BuildBaseURL()
	if err != nil {
		return "", err
	}
	tagOrDigest := ""
	switch v := targetRef.(type) {
	case reference.Tagged:
		tagOrDigest = v.Tag()
	case reference.Digested:
		tagOrDigest = v.Digest().String()
	}
	baseURL = fmt.Sprintf("%s%s/%s/%s", baseURL, reference.Path(targetRef), "manifests", tagOrDigest)
	return baseURL, nil
}

func buildBlobUploadURL(ub *v2.URLBuilder, hostname string, targetRef reference.Named, values url.Values) (string, error) {
	if !isHubLibraryRef(targetRef, hostname) {
		return ub.BuildBlobUploadURL(targetRef, values)
	}
	// this is a library reference and we don't want to lose the "library/" part of the URL ref
	baseURL, err := ub.BuildBaseURL()
	if err != nil {
		return "", err
	}
	baseURL = fmt.Sprintf("%s%s/%s", baseURL, reference.Path(targetRef), "blobs/uploads/")
	return appendValues(baseURL, values), nil
}

func isHubLibraryRef(targetRef reference.Named, hostname string) bool {
	return strings.HasPrefix(reference.Path(targetRef), DefaultRepoPrefix) && hostname == DefaultHostname
}

func getNamedRefWithoutHostname(ref string) (reference.Named, error) {
	targetRef, err := reference.Parse(ref)
	if err != nil {
		return nil, fmt.Errorf("Can't parse reference without hostname: %v", err)
	}
	named, isNamed := targetRef.(reference.Named)
	if !isNamed {
		return nil, fmt.Errorf("Parsed reference is not a Named object: %s", ref)
	}
	return named, nil
}

// NOTE: these two functions are copied from github.com/docker/distribution/registry/api/v2/urls.go
//       to handle the issue of needing to preserve non-normalized names for pushing to "library/" on
//       DockerHub
//
// appendValuesURL appends the parameters to the url.
func appendValuesURL(u *url.URL, values ...url.Values) *url.URL {
	merged := u.Query()

	for _, v := range values {
		for k, vv := range v {
			merged[k] = append(merged[k], vv...)
		}
	}
	u.RawQuery = merged.Encode()
	return u
}

// appendValues appends the parameters to the url. Panics if the string is not
// a url.
func appendValues(u string, values ...url.Values) string {
	up, err := url.Parse(u)

	if err != nil {
		panic(err) // should never happen
	}

	return appendValuesURL(up, values...).String()
}

func statusSuccess(status int) bool {
	return status >= 200 && status <= 399
}
