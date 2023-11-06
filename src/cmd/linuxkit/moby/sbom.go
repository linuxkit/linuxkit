package moby

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

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
}

func NewSbomGenerator(filename string, currentBuildTime bool) (*SbomGenerator, error) {
	if filename == "" {
		return nil, errors.New("filename must be specified")
	}
	buildTime := defaultModTime
	if currentBuildTime {
		buildTime = time.Now()
	}
	return &SbomGenerator{filename, false, nil, buildTime}, nil
}

func (s *SbomGenerator) Add(prefix string, sbom io.ReadCloser) error {
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

	// change any paths to include the prefix
	for i := range doc.Files {
		doc.Files[i].FileName = filepath.Join(prefix, doc.Files[i].FileName)
	}
	for i := range doc.Packages {
		doc.Packages[i].PackageFileName = filepath.Join(prefix, doc.Packages[i].PackageFileName)
		// we should need to add the prefix to each of doc.Packages[i].Files[], but those are pointers,
		// so they point to the actual file structs we handled above
	}
	s.sboms = append(s.sboms, doc)
	return nil
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
