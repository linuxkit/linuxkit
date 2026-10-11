package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	arm64ImageMagicOffset = 0x38
	zbootHeaderSize       = 0x40
	zbootCompressionEnd   = 0x38
	arm64ImageHeaderSize  = 0x40
	linuxPEMagic          = 0x818223cd
	zbootCompressionStart = 0x18
	maxKernelImageSize    = 1 << 30
	maxInt64              = int64(^uint64(0) >> 1)
)

var arm64ImageMagic = [4]byte{'A', 'R', 'M', 0x64}

// prepareVirtualizationFrameworkKernel returns a kernel path the macOS
// Virtualization framework can load. EFI zboot images are containers around a
// compressed kernel, so unwrap them before passing them to the framework.
func prepareVirtualizationFrameworkKernel(path string) (string, func(), error) {
	kernel, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("unable to read kernel file %s: %w", path, err)
	}
	defer func() { _ = kernel.Close() }()

	info, err := kernel.Stat()
	if err != nil {
		return "", nil, fmt.Errorf("unable to stat kernel file %s: %w", path, err)
	}

	prefix := make([]byte, 8)
	n, err := kernel.ReadAt(prefix, 0)
	if err != nil && err != io.EOF {
		return "", nil, fmt.Errorf("unable to read kernel header %s: %w", path, err)
	}
	prefix = prefix[:n]

	if hasZbootMarker(prefix) {
		return writeDecodedKernel(path, func(out io.Writer) error {
			return decodeZbootKernel(kernel, info.Size(), out, maxKernelImageSize)
		})
	}

	if hasGzipMagic(prefix) {
		return writeDecodedKernel(path, func(out io.Writer) error {
			reader, err := gzip.NewReader(kernel)
			if err != nil {
				return fmt.Errorf("unable to read gzip kernel %s: %w", path, err)
			}
			defer func() { _ = reader.Close() }()

			n, err := io.Copy(out, io.LimitReader(reader, maxKernelImageSize+1))
			if err != nil {
				return fmt.Errorf("unable to decompress kernel %s: %w", path, err)
			}
			if n > maxKernelImageSize {
				return fmt.Errorf("decompressed kernel %s exceeds %d bytes", path, maxKernelImageSize)
			}
			return nil
		})
	}

	return path, func() {}, nil
}

func writeDecodedKernel(source string, decode func(io.Writer) error) (string, func(), error) {
	out, err := os.CreateTemp("", "linuxkit-kernel-uncompressed-*")
	if err != nil {
		return "", nil, fmt.Errorf("unable to create decompressed kernel for %s: %w", source, err)
	}
	outPath := out.Name()
	cleanup := func() { _ = os.Remove(outPath) }

	if err := decode(out); err != nil {
		_ = out.Close()
		cleanup()
		return "", nil, fmt.Errorf("unable to decompress kernel %s: %w", source, err)
	}
	if err := out.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("unable to close decompressed kernel for %s: %w", source, err)
	}
	return outPath, cleanup, nil
}

func decodeZbootKernel(image io.ReaderAt, imageSize int64, out io.Writer, maxOutput int64) error {
	if imageSize < zbootHeaderSize {
		return fmt.Errorf("EFI zboot header is truncated: have %d bytes, need %d", imageSize, zbootHeaderSize)
	}

	header := make([]byte, zbootHeaderSize)
	if _, err := image.ReadAt(header, 0); err != nil {
		return fmt.Errorf("unable to read EFI zboot header: %w", err)
	}
	if !bytes.Equal(header[:2], []byte("MZ")) || !bytes.Equal(header[4:8], []byte("zimg")) {
		return fmt.Errorf("invalid EFI zboot signature")
	}

	if binary.LittleEndian.Uint32(header[0x38:0x3c]) != linuxPEMagic {
		return fmt.Errorf("invalid EFI zboot Linux PE signature")
	}

	compressionBytes := header[zbootCompressionStart:zbootCompressionEnd]
	terminator := bytes.IndexByte(compressionBytes, 0)
	if terminator <= 0 {
		return fmt.Errorf("EFI zboot compression name is empty or unterminated")
	}
	compression := string(compressionBytes[:terminator])
	if compression != "gzip" {
		return fmt.Errorf("unsupported EFI zboot compression %q (only gzip is supported)", compression)
	}

	payloadOffset := uint64(binary.LittleEndian.Uint32(header[8:12]))
	payloadSize := uint64(binary.LittleEndian.Uint32(header[12:16]))
	// The Linux zboot build omits the separate four-byte size suffix for gzip;
	// other compression methods append that size word outside payloadSize.
	if payloadOffset < zbootHeaderSize {
		return fmt.Errorf("EFI zboot payload offset %d overlaps the header", payloadOffset)
	}
	if payloadSize == 0 || payloadOffset > uint64(imageSize) || payloadSize > uint64(imageSize)-payloadOffset {
		return fmt.Errorf("EFI zboot payload range [%d, %d) is outside the %d-byte image", payloadOffset, payloadOffset+payloadSize, imageSize)
	}
	if maxOutput < 0 || maxOutput == maxInt64 {
		return fmt.Errorf("invalid maximum kernel size %d", maxOutput)
	}

	payload := io.NewSectionReader(image, int64(payloadOffset), int64(payloadSize))
	reader, err := gzip.NewReader(payload)
	if err != nil {
		return fmt.Errorf("unable to open EFI zboot gzip payload: %w", err)
	}
	defer func() { _ = reader.Close() }()

	validated := &arm64ImageWriter{writer: out}
	n, err := io.Copy(validated, io.LimitReader(reader, maxOutput+1))
	if err != nil {
		return fmt.Errorf("unable to decompress EFI zboot gzip payload: %w", err)
	}
	if n > maxOutput {
		return fmt.Errorf("decompressed EFI zboot kernel exceeds %d bytes", maxOutput)
	}
	if !validated.hasImageMagic() {
		return fmt.Errorf("decompressed EFI zboot payload is not an ARM64 Linux Image")
	}
	return nil
}

type arm64ImageWriter struct {
	writer io.Writer
	prefix [arm64ImageHeaderSize]byte
	seen   int
}

func (w *arm64ImageWriter) Write(p []byte) (int, error) {
	if w.seen < len(w.prefix) {
		n := copy(w.prefix[w.seen:], p)
		w.seen += n
	}
	return w.writer.Write(p)
}

func (w *arm64ImageWriter) hasImageMagic() bool {
	return w.seen == len(w.prefix) && bytes.Equal(w.prefix[arm64ImageMagicOffset:arm64ImageMagicOffset+len(arm64ImageMagic)], arm64ImageMagic[:])
}

func hasZbootMarker(prefix []byte) bool {
	return len(prefix) >= 8 && bytes.Equal(prefix[4:8], []byte("zimg"))
}

func hasGzipMagic(prefix []byte) bool {
	return len(prefix) >= 2 && prefix[0] == 0x1f && prefix[1] == 0x8b
}
