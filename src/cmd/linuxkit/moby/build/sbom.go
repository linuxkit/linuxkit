package build

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd/v2/pkg/reference"
	"github.com/google/uuid"
	spdxjson "github.com/spdx/tools-golang/json"
	"github.com/spdx/tools-golang/spdx"
	spdxcommon "github.com/spdx/tools-golang/spdx/v2/common"
	spdxversion "github.com/spdx/tools-golang/spdx/v2/v2_3"
)

// SbomGenerator handler for generating sbom
type SbomGenerator struct {
	filename  string
	closed    bool
	sboms     []*spdx.Document
	buildTime time.Time
	scopes    map[string]int
}

func NewSbomGenerator(filename string, currentBuildTime bool) (*SbomGenerator, error) {
	if filename == "" {
		return nil, errors.New("filename must be specified")
	}
	buildTime := defaultModTime
	if currentBuildTime {
		buildTime = time.Now()
	}
	return &SbomGenerator{filename, false, nil, buildTime, map[string]int{}}, nil
}

// Add adds the sbom of the image ref, which is unpacked at prefix in the
// filesystem. Every SPDX identifier in it is scoped to the image, so that
// packages and files that several images share remain distinct elements, each
// with its own relationships. The root that the sbom describes is named after
// ref, with prefix as its packageFileName.
func (s *SbomGenerator) Add(prefix string, ref *reference.Spec, sbom io.ReadCloser) error {
	if s.closed {
		return fmt.Errorf("sbom generator already closed")
	}
	doc, err := spdxjson.Read(sbom)
	if err != nil {
		return err
	}
	if err := sbom.Close(); err != nil {
		return err
	}

	scope := spdxIDString(strings.Trim(prefix, "/"))
	if scope == "" {
		scope = spdxIDString(ref.Locator)
	}
	s.scopes[scope]++
	if n := s.scopes[scope]; n > 1 {
		scope = fmt.Sprintf("%s-%d", scope, n)
	}
	scopeIDs(doc, scope)

	roots := map[spdxcommon.ElementID]bool{}
	for _, r := range doc.Relationships {
		if r.Relationship == spdxcommon.TypeRelationshipDescribe && r.RefA.ElementRefID == "DOCUMENT" && r.RefB.DocumentRefID == "" {
			roots[r.RefB.ElementRefID] = true
		}
	}
	tag, _, _ := strings.Cut(ref.Object, "@")
	if tag == "" {
		tag = ref.Digest().String()
	}

	// change any paths to include the prefix
	for i := range doc.Files {
		doc.Files[i].FileName = filepath.Join(prefix, doc.Files[i].FileName)
	}
	for _, p := range doc.Packages {
		switch {
		case roots[p.PackageSPDXIdentifier]:
			p.PackageName = ref.Locator
			p.PackageVersion = tag
			if dir := strings.Trim(prefix, "/"); dir != "" {
				p.PackageFileName = "./" + dir
			}
		case p.PackageFileName != "":
			p.PackageFileName = filepath.Join(prefix, p.PackageFileName)
		}
		// we should need to add the prefix to each of doc.Packages[i].Files[], but those are pointers,
		// so they point to the actual file structs we handled above
	}
	s.sboms = append(s.sboms, doc)
	return nil
}

// spdxIDString replaces each character that an SPDX identifier cannot contain
// with a hyphen.
func spdxIDString(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '.' || r == '-' || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') {
			return r
		}
		return '-'
	}, s)
}

// scopeIDs prefixes every element identifier that doc defines or refers to
// with scope, leaving DOCUMENT and references into other documents unchanged.
func scopeIDs(doc *spdx.Document, scope string) {
	id := func(e spdxcommon.ElementID) spdxcommon.ElementID {
		if e == "" || e == "DOCUMENT" {
			return e
		}
		return spdxcommon.ElementID(scope + "-" + string(e))
	}
	docID := func(e *spdxcommon.DocElementID) {
		if e.DocumentRefID == "" && e.SpecialID == "" {
			e.ElementRefID = id(e.ElementRefID)
		}
	}
	annotations := func(as []spdx.Annotation) {
		for i := range as {
			docID(&as[i].AnnotationSPDXIdentifier)
		}
	}
	seen := map[*spdx.File]bool{}
	file := func(f *spdx.File) {
		if f == nil || seen[f] {
			return
		}
		seen[f] = true
		f.FileSPDXIdentifier = id(f.FileSPDXIdentifier)
		annotations(f.Annotations)
	}
	for _, f := range doc.Files {
		file(f)
	}
	for _, p := range doc.Packages {
		p.PackageSPDXIdentifier = id(p.PackageSPDXIdentifier)
		annotations(p.Annotations)
		for _, f := range p.Files {
			file(f)
		}
	}
	for i := range doc.Snippets {
		sn := &doc.Snippets[i]
		sn.SnippetSPDXIdentifier = id(sn.SnippetSPDXIdentifier)
		sn.SnippetFromFileSPDXIdentifier = id(sn.SnippetFromFileSPDXIdentifier)
	}
	for _, r := range doc.Relationships {
		docID(&r.RefA)
		docID(&r.RefB)
	}
	for _, a := range doc.Annotations {
		docID(&a.AnnotationSPDXIdentifier)
	}
}

// Close finalize generation of the sbom, including merging any together and writing the output file to a tar stream,
// and cleaning up any temporary files.
func (s *SbomGenerator) Close(tw *tar.Writer) error {
	// merge all of the sboms together
	doc := spdx.Document{
		SPDXVersion:       spdxversion.Version,
		DataLicense:       spdxversion.DataLicense,
		DocumentName:      "sbom",
		DocumentNamespace: fmt.Sprintf("https://github.com/linuxkit/linuxkit/sbom-%s", uuid.New().String()),
		CreationInfo: &spdx.CreationInfo{
			LicenseListVersion: "3.20",
			Creators: []spdxcommon.Creator{
				{CreatorType: "Organization", Creator: "LinuxKit"},
				{CreatorType: "Tool", Creator: "linuxkit"},
			},
			Created: s.buildTime.UTC().Format("2006-01-02T15:04:05Z"),
		},
		SPDXIdentifier: spdxcommon.ElementID("DOCUMENT"),
	}
	for _, sbom := range s.sboms {
		doc.Packages = append(doc.Packages, sbom.Packages...)
		doc.Files = append(doc.Files, sbom.Files...)
		doc.OtherLicenses = append(doc.OtherLicenses, sbom.OtherLicenses...)
		doc.Relationships = append(doc.Relationships, sbom.Relationships...)
		doc.Annotations = append(doc.Annotations, sbom.Annotations...)
		doc.ExternalDocumentReferences = append(doc.ExternalDocumentReferences, sbom.ExternalDocumentReferences...)
	}
	var buf bytes.Buffer
	if err := spdxjson.Write(&doc, &buf); err != nil {
		return err
	}
	// create
	hdr := &tar.Header{
		Name:     s.filename,
		Typeflag: tar.TypeReg,
		Mode:     0o644,
		ModTime:  defaultModTime,
		Uid:      int(0),
		Gid:      int(0),
		Format:   tar.FormatPAX,
		Size:     int64(buf.Len()),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := io.Copy(tw, &buf); err != nil && err != io.EOF {
		return fmt.Errorf("failed to write sbom: %v", err)
	}
	s.closed = true
	return nil
}
