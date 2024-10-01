package cache

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
)

func writeLayoutHeader(tw *tar.Writer) error {
	// layout file
	layoutFileBytes := []byte(layoutFile)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "oci-layout",
		Mode:     0644,
		Size:     int64(len(layoutFileBytes)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		return err
	}
	if _, err := tw.Write(layoutFileBytes); err != nil {
		return err
	}

	// make blobs directory
	if err := tw.WriteHeader(&tar.Header{
		Name:     "blobs/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}
	// make blobs/sha256 directory
	if err := tw.WriteHeader(&tar.Header{
		Name:     "blobs/sha256/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}
	return nil
}

func writeLayoutImage(tw *tar.Writer, image v1.Image) error {
	// write config, each layer, manifest, saving the digest for each
	manifest, err := image.Manifest()
	if err != nil {
		return err
	}
	configDesc := manifest.Config
	configBytes, err := image.RawConfigFile()
	if err != nil {
		return err
	}
	if err := writeLayoutBlob(tw, configDesc.Digest.Hex, configDesc.Size, bytes.NewReader(configBytes)); err != nil {
		return err
	}

	layers, err := image.Layers()
	if err != nil {
		return err
	}
	for _, layer := range layers {
		blob, err := layer.Compressed()
		if err != nil {
			return err
		}
		defer blob.Close()
		blobDigest, err := layer.Digest()
		if err != nil {
			return err
		}
		blobSize, err := layer.Size()
		if err != nil {
			return err
		}
		if err := writeLayoutBlob(tw, blobDigest.Hex, blobSize, blob); err != nil {
			return err
		}
	}
	// write the manifest
	manifestSize, err := image.Size()
	if err != nil {
		return err
	}
	manifestDigest, err := image.Digest()
	if err != nil {
		return err
	}
	manifestBytes, err := image.RawManifest()
	if err != nil {
		return err
	}
	if err := writeLayoutBlob(tw, manifestDigest.Hex, manifestSize, bytes.NewReader(manifestBytes)); err != nil {
		return err
	}
	return nil
}

func writeLayoutBlob(tw *tar.Writer, digest string, size int64, blob io.Reader) error {
	if err := tw.WriteHeader(&tar.Header{
		Name:     fmt.Sprintf("blobs/sha256/%s", digest),
		Mode:     0644,
		Size:     size,
		Typeflag: tar.TypeReg,
	}); err != nil {
		return err
	}
	if _, err := io.Copy(tw, blob); err != nil {
		return err
	}
	return nil
}

func writeLayoutIndex(tw *tar.Writer, desc v1.Descriptor) error {
	ii := empty.Index

	index, err := ii.IndexManifest()
	if err != nil {
		return err
	}

	index.Manifests = append(index.Manifests, desc)

	rawIndex, err := json.MarshalIndent(index, "", "   ")
	if err != nil {
		return err
	}
	// write the index
	if err := tw.WriteHeader(&tar.Header{
		Name: "index.json",
		Mode: 0644,
		Size: int64(len(rawIndex)),
	}); err != nil {
		return err
	}
	if _, err := tw.Write(rawIndex); err != nil {
		return err
	}
	return nil
}
