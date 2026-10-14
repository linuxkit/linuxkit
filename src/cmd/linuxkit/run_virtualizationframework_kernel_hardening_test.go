package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	hardeningZbootHeaderSize       = 0x40
	hardeningZbootCompressionStart = 0x18
	hardeningZbootCompressionEnd   = 0x38
	hardeningLinuxMagicOffset      = 0x38
	hardeningPEHeaderOffset        = 0x40
	hardeningPayloadOffset         = 0x100
	hardeningArm64MagicOffset      = 0x38
	hardeningArm64HeaderSize       = 0x40
	hardeningLinuxPEMagic          = 0x818223cd
)

func TestDecodeZbootRejectsPayloadOffsetsInsideFullHeader(t *testing.T) {
	compressed := hardeningGzipBytes(t, hardeningARM64Image())

	for offset := uint32(0x38); offset < hardeningZbootHeaderSize; offset++ {
		t.Run(fmt.Sprintf("offset_%#x", offset), func(t *testing.T) {
			image := hardeningZbootImage(compressed)
			binary.LittleEndian.PutUint32(image[8:12], offset)

			err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), io.Discard, 128)
			if err == nil || !strings.Contains(err.Error(), "overlaps the header") {
				t.Fatalf("decodeZbootKernel() error = %v, want header-overlap error", err)
			}
		})
	}
}

func TestDecodeZbootRejectsMissingLinuxPEMagic(t *testing.T) {
	image := hardeningZbootImage(hardeningGzipBytes(t, hardeningARM64Image()))
	image[hardeningLinuxMagicOffset] ^= 0xff

	err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), io.Discard, 128)
	if err == nil || !strings.Contains(err.Error(), "Linux PE signature") {
		t.Fatalf("decodeZbootKernel() error = %v, want Linux PE signature error", err)
	}
}

func TestDecodeZbootRejectsTruncatedARM64Headers(t *testing.T) {
	fullKernel := hardeningARM64Image()

	for size := hardeningArm64MagicOffset + 4; size < hardeningArm64HeaderSize; size++ {
		t.Run(fmt.Sprintf("decoded_size_%d", size), func(t *testing.T) {
			image := hardeningZbootImage(hardeningGzipBytes(t, fullKernel[:size]))
			err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), io.Discard, 128)
			if err == nil || !strings.Contains(err.Error(), "not an ARM64 Linux Image") {
				t.Fatalf("decodeZbootKernel() error = %v, want truncated ARM64 image error", err)
			}
		})
	}
}

func TestDecodeZbootOutputLimitAtMinimumARM64Header(t *testing.T) {
	image := hardeningZbootImage(hardeningGzipBytes(t, hardeningARM64Image()))

	var got bytes.Buffer
	if err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), &got, hardeningArm64HeaderSize); err != nil {
		t.Fatalf("decodeZbootKernel() at exact 64-byte limit error = %v", err)
	}
	if got.Len() != hardeningArm64HeaderSize {
		t.Fatalf("decoded size = %d, want %d", got.Len(), hardeningArm64HeaderSize)
	}

	err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), io.Discard, hardeningArm64HeaderSize-1)
	if err == nil || !strings.Contains(err.Error(), "exceeds 63 bytes") {
		t.Fatalf("decodeZbootKernel() at 63-byte limit error = %v, want size-limit error", err)
	}
}

func TestDecodeZbootPropagatesReadAndWriteErrors(t *testing.T) {
	image := hardeningZbootImage(hardeningGzipBytes(t, hardeningARM64Image()))
	readFailure := errors.New("injected read failure")
	writeFailure := errors.New("injected write failure")

	t.Run("header read", func(t *testing.T) {
		reader := hardeningFailingReaderAt{data: image, failOffset: 0, err: readFailure}
		err := decodeZbootKernel(reader, int64(len(image)), io.Discard, 128)
		if !errors.Is(err, readFailure) {
			t.Fatalf("decodeZbootKernel() error = %v, want wrapped read failure", err)
		}
	})

	t.Run("payload read", func(t *testing.T) {
		reader := hardeningFailingReaderAt{data: image, failOffset: hardeningPayloadOffset, err: readFailure}
		err := decodeZbootKernel(reader, int64(len(image)), io.Discard, 128)
		if !errors.Is(err, readFailure) {
			t.Fatalf("decodeZbootKernel() error = %v, want wrapped payload read failure", err)
		}
	})

	t.Run("output write", func(t *testing.T) {
		err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), hardeningFailingWriter{err: writeFailure}, 128)
		if !errors.Is(err, writeFailure) {
			t.Fatalf("decodeZbootKernel() error = %v, want wrapped write failure", err)
		}
	})

	t.Run("short output write", func(t *testing.T) {
		err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), hardeningShortWriter{}, 128)
		if !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("decodeZbootKernel() error = %v, want io.ErrShortWrite", err)
		}
	})
}

func TestPrepareKernelTempDirectoryFailureAndDoubleCleanup(t *testing.T) {
	t.Run("temp directory creation failure leaves no artifact", func(t *testing.T) {
		input := filepath.Join(t.TempDir(), "kernel")
		image := hardeningZbootImage(hardeningGzipBytes(t, hardeningARM64Image()))
		if err := os.WriteFile(input, image, 0600); err != nil {
			t.Fatal(err)
		}

		missingTempDir := filepath.Join(t.TempDir(), "does-not-exist")
		t.Setenv("TMPDIR", missingTempDir)
		t.Setenv("TMP", missingTempDir)
		t.Setenv("TEMP", missingTempDir)
		prepared, cleanup, err := prepareVirtualizationFrameworkKernel(input)
		if err == nil {
			if cleanup != nil {
				cleanup()
			}
			t.Fatalf("prepareVirtualizationFrameworkKernel() unexpectedly succeeded with missing TMPDIR: %q", prepared)
		}
		if cleanup != nil {
			t.Fatal("prepareVirtualizationFrameworkKernel() returned cleanup after temp creation failed")
		}
		if _, statErr := os.Stat(missingTempDir); !os.IsNotExist(statErr) {
			t.Fatalf("missing TMPDIR unexpectedly exists: %v", statErr)
		}
	})

	t.Run("cleanup is idempotent", func(t *testing.T) {
		input := filepath.Join(t.TempDir(), "kernel")
		image := hardeningZbootImage(hardeningGzipBytes(t, hardeningARM64Image()))
		if err := os.WriteFile(input, image, 0600); err != nil {
			t.Fatal(err)
		}

		prepared, cleanup, err := prepareVirtualizationFrameworkKernel(input)
		if err != nil {
			t.Fatal(err)
		}
		cleanup()
		cleanup()
		if _, err := os.Stat(prepared); !os.IsNotExist(err) {
			t.Fatalf("temporary kernel still exists after repeated cleanup: %v", err)
		}
	})
}

func FuzzDecodeZbootKernelBoundedOutput(f *testing.F) {
	valid := hardeningZbootImage(hardeningGzipBytes(f, hardeningARM64Image()))
	f.Add(valid)

	for _, cut := range []int{0, 1, 7, 8, 0x37, 0x38, 0x3f, 0x40, hardeningPayloadOffset - 1, hardeningPayloadOffset, len(valid) - 1} {
		if cut >= 0 && cut < len(valid) {
			f.Add(append([]byte(nil), valid[:cut]...))
		}
	}

	f.Fuzz(func(t *testing.T, image []byte) {
		if len(image) > 1<<16 {
			t.Skip()
		}

		const maxOutput = int64(128)
		var got bytes.Buffer
		err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), &got, maxOutput)
		if int64(got.Len()) > maxOutput+1 {
			t.Fatalf("decoder wrote %d bytes with a %d-byte limit", got.Len(), maxOutput)
		}
		if err == nil {
			if int64(got.Len()) > maxOutput {
				t.Fatalf("successful decode wrote %d bytes with a %d-byte limit", got.Len(), maxOutput)
			}
			if got.Len() < hardeningArm64HeaderSize || !bytes.Equal(got.Bytes()[hardeningArm64MagicOffset:hardeningArm64MagicOffset+4], []byte{'A', 'R', 'M', 0x64}) {
				t.Fatalf("successful decode returned an invalid or undersized ARM64 image")
			}
		}
	})
}

type hardeningFailingReaderAt struct {
	data       []byte
	failOffset int64
	err        error
}

func (r hardeningFailingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off == r.failOffset {
		return 0, r.err
	}
	return bytes.NewReader(r.data).ReadAt(p, off)
}

type hardeningFailingWriter struct {
	err error
}

func (w hardeningFailingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type hardeningShortWriter struct{}

func (hardeningShortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func hardeningARM64Image() []byte {
	image := make([]byte, hardeningArm64HeaderSize)
	copy(image[hardeningArm64MagicOffset:], []byte{'A', 'R', 'M', 0x64})
	return image
}

func hardeningGzipBytes(t testing.TB, payload []byte) []byte {
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

func hardeningZbootImage(payload []byte) []byte {
	image := make([]byte, hardeningPayloadOffset+len(payload))
	copy(image[:2], "MZ")
	copy(image[4:8], "zimg")
	binary.LittleEndian.PutUint32(image[8:12], hardeningPayloadOffset)
	binary.LittleEndian.PutUint32(image[12:16], uint32(len(payload)))
	copy(image[hardeningZbootCompressionStart:hardeningZbootCompressionEnd], "gzip")
	binary.LittleEndian.PutUint32(image[hardeningLinuxMagicOffset:hardeningLinuxMagicOffset+4], hardeningLinuxPEMagic)
	binary.LittleEndian.PutUint32(image[0x3c:0x40], hardeningPEHeaderOffset)
	copy(image[hardeningPEHeaderOffset:hardeningPEHeaderOffset+4], "PE\x00\x00")
	copy(image[hardeningPayloadOffset:], payload)
	return image
}
