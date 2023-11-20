package pool

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/klauspost/compress/flate"
)

const DefaultBufferSize = 2 * 1024 * 1024

var FilePool = sync.Pool{
	New: func() any {
		return &File{CompressedData: bytes.NewBuffer(make([]byte, DefaultBufferSize))}
	},
}

// A File refers to a file-backed buffer
type File struct {
	Info           fs.FileInfo
	Header         *zip.FileHeader
	CompressedData *bytes.Buffer
	Overflow       *os.File
	Compressor     *flate.Writer
	Path           string
	written        int64
}

func NewFile(path string, info fs.FileInfo, relativeTo string) (*File, error) {
	f := FilePool.Get().(*File)
	err := f.Reset(path, info, relativeTo)
	return f, err
}

// Reset resets the file-backed buffer ready to be used by another file.
func (f *File) Reset(path string, info fs.FileInfo, relativeTo string) error {
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("file info header for %q: %w", path, err)
	}
	f.Path = path
	f.Info = info
	f.Header = hdr
	f.CompressedData.Reset()
	f.Overflow = nil
	f.written = 0

	if f.Compressor == nil {
		f.Compressor, err = flate.NewWriter(f, flate.DefaultCompression)
		if err != nil {
			return fmt.Errorf("new compressor: %w", err)
		}
	} else {
		f.Compressor.Reset(f)
	}

	if relativeTo != "" {
		if err := f.setNameRelativeTo(relativeTo); err != nil {
			return fmt.Errorf("set name relative to %q: %w", relativeTo, err)
		}
	}

	return nil
}

func (f *File) Write(p []byte) (n int, err error) {
	if f.CompressedData.Available() != 0 {
		maxWriteable := min(f.CompressedData.Available(), len(p))
		f.written += int64(maxWriteable)
		f.CompressedData.Write(p[:maxWriteable])
		p = p[maxWriteable:]
	}

	if len(p) > 0 {
		if f.Overflow == nil {
			if f.Overflow, err = os.CreateTemp("", "pzip-overflow"); err != nil {
				return len(p), fmt.Errorf("create temporary file: %w", err)
			}
		}

		if _, err := f.Overflow.Write(p); err != nil {
			return len(p), fmt.Errorf("write temporary file for %q: %w", f.Header.Name, err)
		}
		f.written += int64(len(p))
	}

	return len(p), nil
}

// Written returns the number of bytes of the file compressed and written to a destination
func (f *File) Written() int64 {
	return f.written
}

// Overflowed returns true if the compressed contents of the file was too large to fit in the in-memory buffer.
// The overflowed contents are written to a temporary file.
func (f *File) Overflowed() bool {
	return f.Overflow != nil
}

func (f *File) setNameRelativeTo(root string) error {
	relativeToRoot, err := filepath.Rel(root, f.Path)
	if err != nil {
		return fmt.Errorf("relative path of %q to root %q: %w", f.Path, root, err)
	}
	f.Header.Name = filepath.Join(filepath.Base(root), relativeToRoot)
	return nil
}
