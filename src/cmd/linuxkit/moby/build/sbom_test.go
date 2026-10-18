package build

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/containerd/containerd/v2/pkg/reference"
)

// scannedSBoM is an SPDX document of the shape the buildkit syft scanner
// attests for every image, so the same identifiers recur in each image.
const scannedSBoM = `{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "sbom",
  "documentNamespace": "https://example.com/sbom",
  "creationInfo": {"created": "2026-01-01T00:00:00Z", "creators": ["Tool: syft"]},
  "documentDescribes": ["SPDXRef-DocumentRoot-Directory-sbom"],
  "packages": [
    {"SPDXID": "SPDXRef-DocumentRoot-Directory-sbom", "name": "sbom", "downloadLocation": "NOASSERTION", "filesAnalyzed": false},
    {"SPDXID": "SPDXRef-Package-apk-busybox-1", "name": "busybox", "versionInfo": "1.37.0-r19", "downloadLocation": "NOASSERTION",
     "filesAnalyzed": false, "hasFiles": ["SPDXRef-File-bin-busybox-2"]}
  ],
  "files": [
    {"SPDXID": "SPDXRef-File-bin-busybox-2", "fileName": "/bin/busybox",
     "checksums": [{"algorithm": "SHA256", "checksumValue": "00"}]}
  ],
  "relationships": [
    {"spdxElementId": "SPDXRef-DocumentRoot-Directory-sbom", "relationshipType": "CONTAINS", "relatedSpdxElement": "SPDXRef-Package-apk-busybox-1"},
    {"spdxElementId": "SPDXRef-Package-apk-busybox-1", "relationshipType": "DEPENDENCY_OF", "relatedSpdxElement": "NOASSERTION"}
  ]
}`

type testSBoM struct {
	DocumentDescribes []string `json:"documentDescribes"`
	Packages          []struct {
		SPDXID          string `json:"SPDXID"`
		Name            string `json:"name"`
		VersionInfo     string `json:"versionInfo"`
		PackageFileName string `json:"packageFileName"`
	} `json:"packages"`
	Files []struct {
		SPDXID   string `json:"SPDXID"`
		FileName string `json:"fileName"`
	} `json:"files"`
	Relationships []struct {
		A    string `json:"spdxElementId"`
		Type string `json:"relationshipType"`
		B    string `json:"relatedSpdxElement"`
	} `json:"relationships"`
}

func mergeSBoMs(t *testing.T, images [][2]string) testSBoM {
	t.Helper()
	g, err := NewSbomGenerator("sbom.spdx.json", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, image := range images {
		ref, err := reference.Parse(image[1])
		if err != nil {
			t.Fatal(err)
		}
		if err := g.Add(image[0], &ref, io.NopCloser(strings.NewReader(scannedSBoM))); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := g.Close(tw); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(&buf)
	if _, err := tr.Next(); err != nil {
		t.Fatal(err)
	}
	var doc testSBoM
	if err := json.NewDecoder(tr).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestSbomMergeScopesIDs(t *testing.T) {
	doc := mergeSBoMs(t, [][2]string{
		{"", "docker.io/lfedge/eve-init:0123abcd"},
		{"", "docker.io/lfedge/eve-init:0123abcd"},
		{"containers/services/pillar/lower/", "docker.io/lfedge/eve-pillar:4567ef01@sha256:" + strings.Repeat("a", 64)},
	})

	ids := map[string]bool{"SPDXRef-DOCUMENT": true}
	for _, p := range doc.Packages {
		if ids[p.SPDXID] {
			t.Errorf("duplicate SPDX ID %s", p.SPDXID)
		}
		ids[p.SPDXID] = true
	}
	for _, f := range doc.Files {
		if ids[f.SPDXID] {
			t.Errorf("duplicate SPDX ID %s", f.SPDXID)
		}
		ids[f.SPDXID] = true
	}
	if len(ids) != 1+3*3 {
		t.Errorf("got %d SPDX IDs, want %d", len(ids), 1+3*3)
	}
	contains := map[string]bool{}
	for _, r := range doc.Relationships {
		for _, id := range []string{r.A, r.B} {
			if id != "NOASSERTION" && !ids[id] {
				t.Errorf("relationship %s %s %s refers to an undefined element", r.A, r.Type, r.B)
			}
		}
		if r.Type == "CONTAINS" {
			contains[r.A+" "+r.B] = true
		}
	}

	roots := map[string]string{
		"SPDXRef-docker.io-lfedge-eve-init-DocumentRoot-Directory-sbom":        "docker.io/lfedge/eve-init 0123abcd ",
		"SPDXRef-docker.io-lfedge-eve-init-2-DocumentRoot-Directory-sbom":      "docker.io/lfedge/eve-init 0123abcd ",
		"SPDXRef-containers-services-pillar-lower-DocumentRoot-Directory-sbom": "docker.io/lfedge/eve-pillar 4567ef01 ./containers/services/pillar/lower",
	}
	for _, p := range doc.Packages {
		if want, ok := roots[p.SPDXID]; ok {
			if got := p.Name + " " + p.VersionInfo + " " + p.PackageFileName; got != want {
				t.Errorf("root %s is %q, want %q", p.SPDXID, got, want)
			}
			continue
		}
		if p.PackageFileName != "" {
			t.Errorf("package %s has packageFileName %s", p.SPDXID, p.PackageFileName)
		}
	}
	describes := map[string]bool{}
	for _, r := range doc.Relationships {
		if r.A == "SPDXRef-DOCUMENT" && r.Type == "DESCRIBES" {
			describes[r.B] = true
		}
	}
	for _, id := range doc.DocumentDescribes {
		describes[id] = true
	}
	for root := range roots {
		if !describes[root] {
			t.Errorf("document does not describe %s", root)
		}
	}
	if !contains["SPDXRef-containers-services-pillar-lower-DocumentRoot-Directory-sbom SPDXRef-containers-services-pillar-lower-Package-apk-busybox-1"] {
		t.Errorf("pillar root does not contain its own busybox")
	}

	files := map[string]bool{}
	for _, f := range doc.Files {
		files[f.SPDXID+" "+f.FileName] = true
	}
	for _, want := range []string{
		"SPDXRef-docker.io-lfedge-eve-init-File-bin-busybox-2 /bin/busybox",
		"SPDXRef-containers-services-pillar-lower-File-bin-busybox-2 containers/services/pillar/lower/bin/busybox",
	} {
		if !files[want] {
			t.Errorf("no file %s in %v", want, files)
		}
	}
}

func TestSpdxIDString(t *testing.T) {
	for in, want := range map[string]string{
		"containers/onboot/000-rngd/rootfs": "containers-onboot-000-rngd-rootfs",
		"docker.io/lfedge/eve-init":         "docker.io-lfedge-eve-init",
		"a_b:c@d":                           "a-b-c-d",
	} {
		if got := spdxIDString(in); got != want {
			t.Errorf("spdxIDString(%q) = %q, want %q", in, got, want)
		}
	}
}
