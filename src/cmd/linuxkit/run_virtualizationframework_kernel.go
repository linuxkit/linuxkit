package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
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
	return prepareVirtualizationFrameworkKernelContext(context.Background(), path)
}

func prepareVirtualizationFrameworkKernelContext(ctx context.Context, path string) (string, func(), error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	kernel, err := openVirtualizationFrameworkKernelFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("unable to read kernel file %s: %w", path, err)
	}
	defer func() { _ = kernel.Close() }()

	info, err := kernel.Stat()
	if err != nil {
		return "", nil, fmt.Errorf("unable to stat kernel file %s: %w", path, err)
	}

	prefix := make([]byte, 8)
	n, err := (contextReaderAt{ctx: ctx, reader: kernel}).ReadAt(prefix, 0)
	if err != nil && err != io.EOF {
		return "", nil, fmt.Errorf("unable to read kernel header %s: %w", path, err)
	}
	prefix = prefix[:n]

	if hasZbootMarker(prefix) {
		return writeDecodedKernelContext(ctx, path, func(out io.Writer) error {
			return decodeZbootKernelContext(ctx, kernel, info.Size(), out, maxKernelImageSize)
		})
	}

	if hasGzipMagic(prefix) {
		return writeDecodedKernelContext(ctx, path, func(out io.Writer) error {
			return decodeGzipKernelContext(ctx, kernel, out, path, maxKernelImageSize)
		})
	}

	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	return path, func() {}, nil
}

func writeDecodedKernelContext(ctx context.Context, source string, decode func(io.Writer) error) (string, func(), error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	out, err := os.CreateTemp("", "linuxkit-kernel-uncompressed-*")
	if err != nil {
		return "", nil, fmt.Errorf("unable to create decompressed kernel for %s: %w", source, err)
	}
	outPath := out.Name()
	cleanup := func() {
		if err := removeDecodedKernelTemp(outPath); err != nil {
			log.Printf("unable to remove decompressed kernel %s: %v", outPath, err)
		}
	}

	if err := decode(contextWriter{ctx: ctx, writer: out}); err != nil {
		cleanupErr := closeAndRemoveDecodedKernelTemp(out, outPath)
		return "", nil, fmt.Errorf("unable to decompress kernel %s: %w", source, errors.Join(err, cleanupErr))
	}
	if err := ctx.Err(); err != nil {
		cleanupErr := closeAndRemoveDecodedKernelTemp(out, outPath)
		return "", nil, fmt.Errorf("unable to decompress kernel %s: %w", source, errors.Join(err, cleanupErr))
	}
	if err := out.Close(); err != nil {
		cleanupErr := removeDecodedKernelTemp(outPath)
		return "", nil, fmt.Errorf("unable to close decompressed kernel for %s: %w", source, errors.Join(err, cleanupErr))
	}
	if err := ctx.Err(); err != nil {
		cleanupErr := removeDecodedKernelTemp(outPath)
		return "", nil, fmt.Errorf("unable to decompress kernel %s: %w", source, errors.Join(err, cleanupErr))
	}
	return outPath, cleanup, nil
}

func closeAndRemoveDecodedKernelTemp(out *os.File, path string) error {
	return errors.Join(out.Close(), removeDecodedKernelTemp(path))
}

func removeDecodedKernelTemp(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func decodeZbootKernel(image io.ReaderAt, imageSize int64, out io.Writer, maxOutput int64) error {
	return decodeZbootKernelContext(context.Background(), image, imageSize, out, maxOutput)
}

func decodeZbootKernelContext(ctx context.Context, image io.ReaderAt, imageSize int64, out io.Writer, maxOutput int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if imageSize < zbootHeaderSize {
		return fmt.Errorf("EFI zboot header is truncated: have %d bytes, need %d", imageSize, zbootHeaderSize)
	}

	header := make([]byte, zbootHeaderSize)
	contextImage := contextReaderAt{ctx: ctx, reader: image}
	if _, err := contextImage.ReadAt(header, 0); err != nil {
		return fmt.Errorf("unable to read EFI zboot header: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
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

	payload := io.NewSectionReader(contextImage, int64(payloadOffset), int64(payloadSize))
	reader, err := gzip.NewReader(payload)
	if err != nil {
		return fmt.Errorf("unable to open EFI zboot gzip payload: %w", err)
	}
	defer func() { _ = reader.Close() }()
	if err := ctx.Err(); err != nil {
		return err
	}

	validated := &arm64ImageWriter{writer: contextWriter{ctx: ctx, writer: out}}
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
	return ctx.Err()
}

func decodeGzipKernelContext(ctx context.Context, input io.Reader, out io.Writer, source string, maxOutput int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if maxOutput < 0 || maxOutput == maxInt64 {
		return fmt.Errorf("invalid maximum kernel size %d", maxOutput)
	}
	reader, err := gzip.NewReader(contextReader{ctx: ctx, reader: input})
	if err != nil {
		return fmt.Errorf("unable to read gzip kernel %s: %w", source, err)
	}
	defer func() { _ = reader.Close() }()
	if err := ctx.Err(); err != nil {
		return err
	}

	n, err := io.Copy(contextWriter{ctx: ctx, writer: out}, io.LimitReader(reader, maxOutput+1))
	if err != nil {
		return fmt.Errorf("unable to decompress kernel %s: %w", source, err)
	}
	if n > maxOutput {
		return fmt.Errorf("decompressed kernel %s exceeds %d bytes", source, maxOutput)
	}
	return ctx.Err()
}

type contextReaderAt struct {
	ctx    context.Context
	reader io.ReaderAt
}

func (r contextReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.ReadAt(p, off)
	if contextErr := r.ctx.Err(); contextErr != nil {
		return n, contextErr
	}
	return n, err
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if contextErr := r.ctx.Err(); contextErr != nil {
		return n, contextErr
	}
	return n, err
}

type contextWriter struct {
	ctx    context.Context
	writer io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.writer.Write(p)
	if contextErr := w.ctx.Err(); contextErr != nil {
		return n, contextErr
	}
	return n, err
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
