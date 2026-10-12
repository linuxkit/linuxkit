package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeGzipKernelContextRejectsMalformedStreams(t *testing.T) {
	valid := gzipBytes(t, cancellationTestARM64Image(4096))
	truncated := append([]byte(nil), valid[:len(valid)-1]...)
	badChecksum := append([]byte(nil), valid...)
	badChecksum[len(badChecksum)-8] ^= 0xff

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "invalid header", input: []byte("not gzip")},
		{name: "truncated trailer", input: truncated},
		{name: "corrupt checksum", input: badChecksum},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decodeGzipKernelContext(context.Background(), bytes.NewReader(tt.input), io.Discard, "Image.gz", 1<<20)
			if err == nil {
				t.Fatal("decodeGzipKernelContext() unexpectedly accepted malformed gzip data")
			}
			if tt.name == "corrupt checksum" && !errors.Is(err, gzip.ErrChecksum) {
				t.Fatalf("decodeGzipKernelContext() error = %v, want gzip checksum error", err)
			}
			if tt.name == "truncated trailer" && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("decodeGzipKernelContext() error = %v, want unexpected EOF", err)
			}
		})
	}
}

func TestDecodeGzipKernelContextRejectsShortWrites(t *testing.T) {
	input := gzipBytes(t, cancellationTestARM64Image(4096))
	err := decodeGzipKernelContext(context.Background(), bytes.NewReader(input), gzipShortWriter{}, "Image.gz", 1<<20)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("decodeGzipKernelContext() error = %v, want io.ErrShortWrite", err)
	}
}

func TestDecodeGzipKernelContextValidatesOutputBounds(t *testing.T) {
	// A valid header with no trailer would otherwise be accepted if an overflowed
	// or negative limit made io.LimitReader return EOF without reading the body.
	input := gzipBytes(t, cancellationTestARM64Image(4096))[:10]
	for _, tt := range []struct {
		name      string
		maxOutput int64
	}{
		{name: "negative", maxOutput: -1},
		{name: "max-int64 overflow", maxOutput: maxInt64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := decodeGzipKernelContext(context.Background(), bytes.NewReader(input), io.Discard, "Image.gz", tt.maxOutput)
			if err == nil || !strings.Contains(err.Error(), "invalid maximum kernel size") {
				t.Fatalf("decodeGzipKernelContext() error = %v, want invalid maximum-size error", err)
			}
		})
	}
}

func TestDecodeGzipKernelContextPreservesConcatenatedMembersAndCombinedLimit(t *testing.T) {
	first := cancellationTestARM64Image(256)
	second := []byte("second gzip member payload")
	want := append(append([]byte(nil), first...), second...)
	compressed := append(gzipBytes(t, first), gzipBytes(t, second)...)

	t.Run("plain gzip", func(t *testing.T) {
		var got bytes.Buffer
		if err := decodeGzipKernelContext(context.Background(), bytes.NewReader(compressed), &got, "Image.gz", int64(len(want))); err != nil {
			t.Fatalf("decodeGzipKernelContext() rejected concatenated members: %v", err)
		}
		if !bytes.Equal(got.Bytes(), want) {
			t.Fatal("concatenated members did not produce combined output")
		}

		err := decodeGzipKernelContext(context.Background(), bytes.NewReader(compressed), io.Discard, "Image.gz", int64(len(want)-1))
		if err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("decodeGzipKernelContext() error = %v, want combined output limit error", err)
		}
	})

	t.Run("zboot gzip", func(t *testing.T) {
		image := testZbootImage(t, "gzip", compressed)
		var got bytes.Buffer
		if err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), &got, int64(len(want))); err != nil {
			t.Fatalf("decodeZbootKernel() rejected concatenated members: %v", err)
		}
		if !bytes.Equal(got.Bytes(), want) {
			t.Fatal("concatenated members did not produce combined output")
		}

		err := decodeZbootKernel(bytes.NewReader(image), int64(len(image)), io.Discard, int64(len(want)-1))
		if err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("decodeZbootKernel() error = %v, want combined output limit error", err)
		}
	})
}

func TestDecodedKernelTempCleanupLogsFailureAndIgnoresMissingFile(t *testing.T) {
	workDir := t.TempDir()
	t.Setenv("TMPDIR", workDir)

	var logs bytes.Buffer
	previousLogOutput := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previousLogOutput)

	prepared, cleanup, err := writeDecodedKernelContext(context.Background(), "test kernel", func(out io.Writer) error {
		_, err := out.Write([]byte("kernel"))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(prepared); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(prepared, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prepared, "child"), []byte("keep directory nonempty"), 0600); err != nil {
		t.Fatal(err)
	}

	cleanup()
	if !strings.Contains(logs.String(), "unable to remove decompressed kernel") {
		t.Fatalf("cleanup failure was not logged: %q", logs.String())
	}
	if err := os.RemoveAll(prepared); err != nil {
		t.Fatal(err)
	}

	logs.Reset()
	if err := removeDecodedKernelTemp(filepath.Join(workDir, "already-removed")); err != nil {
		t.Fatalf("removeDecodedKernelTemp() for missing path error = %v, want nil", err)
	}
	if logs.Len() != 0 {
		t.Fatalf("missing-file cleanup unexpectedly logged: %q", logs.String())
	}
}

func TestCloseAndRemoveDecodedKernelTempSurfacesRemovalFailure(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "kernel-temp-*")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "child"), []byte("keep directory nonempty"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(path); err != nil {
			t.Errorf("remove temporary kernel directory: %v", err)
		}
	})

	if err := closeAndRemoveDecodedKernelTemp(file, path); err == nil {
		t.Fatal("closeAndRemoveDecodedKernelTemp() hid a failed removal")
	}
}

type gzipShortWriter struct{}

func (gzipShortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}
