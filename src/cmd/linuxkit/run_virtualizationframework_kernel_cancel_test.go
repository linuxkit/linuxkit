package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareVirtualizationFrameworkKernelContextCancelsDecodeAndRecovers(t *testing.T) {
	tests := []struct {
		name      string
		makeInput func(*testing.T, []byte) ([]byte, func(context.Context, []byte, *cancelAfterWritesWriter) error)
	}{
		{
			name: "zboot gzip",
			makeInput: func(t *testing.T, kernel []byte) ([]byte, func(context.Context, []byte, *cancelAfterWritesWriter) error) {
				image := testZbootImage(t, "gzip", gzipBytes(t, kernel))
				return image, func(ctx context.Context, input []byte, out *cancelAfterWritesWriter) error {
					return decodeZbootKernelContext(ctx, bytes.NewReader(input), int64(len(input)), out, maxKernelImageSize)
				}
			},
		},
		{
			name: "plain gzip",
			makeInput: func(t *testing.T, kernel []byte) ([]byte, func(context.Context, []byte, *cancelAfterWritesWriter) error) {
				image := gzipBytes(t, kernel)
				return image, func(ctx context.Context, input []byte, out *cancelAfterWritesWriter) error {
					return decodeGzipKernelContext(ctx, bytes.NewReader(input), out, "test kernel", maxKernelImageSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			tempDir := filepath.Join(workDir, "temporary")
			if err := os.Mkdir(tempDir, 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("TMPDIR", tempDir)

			kernel := cancellationTestARM64Image(1 << 20)
			compressedInput, decode := tt.makeInput(t, kernel)
			inputPath := filepath.Join(workDir, "kernel")
			if err := os.WriteFile(inputPath, compressedInput, 0600); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			output := &cancelAfterWritesWriter{cancel: cancel, cancelAfter: 2}
			_, _, err := writeDecodedKernelContext(ctx, inputPath, func(out io.Writer) error {
				output.writer = out
				return decode(ctx, compressedInput, output)
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("writeDecodedKernelContext() error = %v, want context cancellation", err)
			}
			if output.writes != output.cancelAfter || output.bytesWritten == 0 {
				t.Fatalf("cancellation happened before partial output: writes=%d bytes=%d", output.writes, output.bytesWritten)
			}

			entries, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("canceled decode left temporary files behind: %v", entries)
			}

			prepared, cleanup, err := prepareVirtualizationFrameworkKernelContext(context.Background(), inputPath)
			if err != nil {
				t.Fatalf("next normal preparation failed after cancellation: %v", err)
			}
			defer cleanup()
			got, err := os.ReadFile(prepared)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, kernel) {
				t.Fatal("next normal preparation returned incorrect kernel data")
			}
		})
	}
}

func TestPrepareVirtualizationFrameworkKernelContextChecksBeforePassThrough(t *testing.T) {
	input := filepath.Join(t.TempDir(), "Image")
	if err := os.WriteFile(input, testARM64Image(), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	prepared, cleanup, err := prepareVirtualizationFrameworkKernelContext(ctx, input)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("prepareVirtualizationFrameworkKernelContext() error = %v, want context cancellation", err)
	}
	if prepared != "" || cleanup != nil {
		t.Fatalf("canceled pass-through returned prepared=%q cleanup=%v", prepared, cleanup != nil)
	}
}

func cancellationTestARM64Image(size int) []byte {
	image := make([]byte, size)
	copy(image, testARM64Image())
	for i := arm64ImageHeaderSize; i < len(image); i++ {
		image[i] = byte((i*31 + i/7) % 251)
	}
	return image
}

type cancelAfterWritesWriter struct {
	writer       io.Writer
	cancel       context.CancelFunc
	cancelAfter  int
	writes       int
	bytesWritten int
}

func (w *cancelAfterWritesWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err != nil {
		return n, err
	}
	w.writes++
	w.bytesWritten += n
	if w.writes == w.cancelAfter {
		w.cancel()
	}
	return n, nil
}
