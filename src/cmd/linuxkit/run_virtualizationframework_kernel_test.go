package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeZbootKernel(t *testing.T) {
	want := testARM64Image()
	image := testZbootImage(t, "gzip", gzipBytes(t, want))

	var got bytes.Buffer
	if err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), &got, 1024); err != nil {
		t.Fatalf("decodeZbootKernel() error = %v", err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatal("decoded kernel does not match the gzip payload")
	}
}

func TestDecodeZbootKernelRejectsMalformedImages(t *testing.T) {
	compressed := gzipBytes(t, testARM64Image())
	valid := testZbootImage(t, "gzip", compressed)

	noTerminator := testZbootImage(t, "gzip", compressed)
	for i := zbootCompressionStart; i < zbootCompressionEnd; i++ {
		noTerminator[i] = 'x'
	}

	outOfBounds := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(outOfBounds[8:12], uint32(len(outOfBounds)-1))
	binary.LittleEndian.PutUint32(outOfBounds[12:16], 2)

	overlappingPayload := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(overlappingPayload[8:12], zbootHeaderSize-1)

	badSignature := append([]byte(nil), valid...)
	badSignature[0] = 'X'

	badKernel := testZbootImage(t, "gzip", gzipBytes(t, make([]byte, 64)))
	badGzip := testZbootImage(t, "gzip", []byte("not gzip"))
	truncatedGzip := testZbootImage(t, "gzip", compressed[:len(compressed)-4])
	corruptGzip := append([]byte(nil), compressed...)
	corruptGzip[len(corruptGzip)-8] ^= 0xff
	corruptGzip = testZbootImage(t, "gzip", corruptGzip)
	unsupported := testZbootImage(t, "lz4", compressed)

	tests := []struct {
		name      string
		image     []byte
		maxOutput int64
		wantError string
	}{
		{name: "truncated header", image: valid[:zbootHeaderSize-1], maxOutput: 1024, wantError: "truncated"},
		{name: "invalid signature", image: badSignature, maxOutput: 1024, wantError: "signature"},
		{name: "overlapping payload", image: overlappingPayload, maxOutput: 1024, wantError: "overlaps"},
		{name: "payload outside file", image: outOfBounds, maxOutput: 1024, wantError: "outside"},
		{name: "unterminated compression name", image: noTerminator, maxOutput: 1024, wantError: "unterminated"},
		{name: "unsupported compression", image: unsupported, maxOutput: 1024, wantError: "only gzip is supported"},
		{name: "invalid gzip payload", image: badGzip, maxOutput: 1024, wantError: "gzip payload"},
		{name: "truncated gzip payload", image: truncatedGzip, maxOutput: 1024, wantError: "decompress EFI zboot gzip payload"},
		{name: "corrupt gzip checksum", image: corruptGzip, maxOutput: 1024, wantError: "decompress EFI zboot gzip payload"},
		{name: "non-kernel payload", image: badKernel, maxOutput: 1024, wantError: "not an ARM64 Linux Image"},
		{name: "output limit", image: valid, maxOutput: 32, wantError: "exceeds 32 bytes"},
		{name: "maximum integer output limit", image: valid, maxOutput: maxInt64, wantError: "invalid maximum kernel size"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decodeZbootKernel(bytes.NewReader(tt.image), int64(len(tt.image)), io.Discard, tt.maxOutput)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("decodeZbootKernel() error = %v, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestPrepareVirtualizationFrameworkKernel(t *testing.T) {
	t.Run("zboot image is unpacked and cleaned", func(t *testing.T) {
		want := testARM64Image()
		input := filepath.Join(t.TempDir(), "kernel")
		if err := os.WriteFile(input, testZbootImage(t, "gzip", gzipBytes(t, want)), 0600); err != nil {
			t.Fatal(err)
		}

		prepared, cleanup, err := prepareVirtualizationFrameworkKernel(input)
		if err != nil {
			t.Fatal(err)
		}
		if prepared == input {
			t.Fatal("EFI zboot wrapper was not unpacked")
		}
		got, err := os.ReadFile(prepared)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatal("prepared kernel does not match the compressed payload")
		}
		cleanup()
		if _, err := os.Stat(prepared); !os.IsNotExist(err) {
			t.Fatalf("temporary kernel still exists after cleanup: %v", err)
		}
	})

	t.Run("raw kernel is passed through", func(t *testing.T) {
		input := filepath.Join(t.TempDir(), "Image")
		if err := os.WriteFile(input, testARM64Image(), 0600); err != nil {
			t.Fatal(err)
		}

		prepared, cleanup, err := prepareVirtualizationFrameworkKernel(input)
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		if prepared != input {
			t.Fatalf("prepared path = %q, want raw input path %q", prepared, input)
		}
	})

	t.Run("ARM64 EFI-stub kernel with MZ prefix is passed through", func(t *testing.T) {
		image := testARM64Image()
		copy(image, "MZ")
		input := filepath.Join(t.TempDir(), "Image")
		if err := os.WriteFile(input, image, 0600); err != nil {
			t.Fatal(err)
		}

		prepared, cleanup, err := prepareVirtualizationFrameworkKernel(input)
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		if prepared != input {
			t.Fatalf("prepared path = %q, want raw input path %q", prepared, input)
		}
	})

	t.Run("ordinary PE image with MZ prefix is passed through", func(t *testing.T) {
		image := make([]byte, 0x40)
		copy(image[:2], "MZ")
		copy(image[0x3c:0x40], "PE\x00\x00")
		input := filepath.Join(t.TempDir(), "bzImage")
		if err := os.WriteFile(input, image, 0600); err != nil {
			t.Fatal(err)
		}

		prepared, cleanup, err := prepareVirtualizationFrameworkKernel(input)
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		if prepared != input {
			t.Fatalf("prepared path = %q, want raw input path %q", prepared, input)
		}
	})

	t.Run("plain gzip kernel remains supported", func(t *testing.T) {
		want := testARM64Image()
		input := filepath.Join(t.TempDir(), "Image.gz")
		if err := os.WriteFile(input, gzipBytes(t, want), 0600); err != nil {
			t.Fatal(err)
		}

		prepared, cleanup, err := prepareVirtualizationFrameworkKernel(input)
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		got, err := os.ReadFile(prepared)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatal("plain gzip kernel was not preserved after decompression")
		}
	})

	t.Run("failed zboot decode removes temporary output", func(t *testing.T) {
		workDir := t.TempDir()
		tempDir := filepath.Join(workDir, "tmp")
		if err := os.Mkdir(tempDir, 0700); err != nil {
			t.Fatal(err)
		}
		input := filepath.Join(workDir, "kernel")
		if err := os.WriteFile(input, testZbootImage(t, "gzip", []byte("broken gzip")), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("TMPDIR", tempDir)

		if _, _, err := prepareVirtualizationFrameworkKernel(input); err == nil {
			t.Fatal("prepareVirtualizationFrameworkKernel() unexpectedly accepted a corrupt payload")
		}
		entries, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("failed decode left temporary files behind: %v", entries)
		}
	})
}

func testARM64Image() []byte {
	image := make([]byte, arm64ImageHeaderSize)
	copy(image[arm64ImageMagicOffset:], arm64ImageMagic[:])
	return image
}

func gzipBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return compressed.Bytes()
}

func testZbootImage(t *testing.T, compression string, payload []byte) []byte {
	t.Helper()
	const payloadOffset = 0x100
	image := make([]byte, payloadOffset+len(payload))
	copy(image[:2], "MZ")
	copy(image[4:8], "zimg")
	binary.LittleEndian.PutUint32(image[8:12], payloadOffset)
	binary.LittleEndian.PutUint32(image[12:16], uint32(len(payload)))
	copy(image[zbootCompressionStart:zbootCompressionEnd], compression)
	binary.LittleEndian.PutUint32(image[0x38:0x3c], linuxPEMagic)
	copy(image[payloadOffset:], payload)
	return image
}
