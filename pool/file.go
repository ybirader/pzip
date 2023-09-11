package pool

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/klauspost/compress/flate"
	"github.com/pkg/errors"
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
		return errors.Errorf("ERROR: could not get file info header for %s: %v", path, err)
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
			return errors.New("ERROR: could not create compressor")
		}
	} else {
		f.Compressor.Reset(f)
	}

	if relativeTo != "" {
		f.setNameRelativeTo(relativeTo)
	}

	return nil
}

func (f *File) Write(p []byte) (n int, err error) {
	if f.CompressedData.Available() != 0 {
		maxWritable := min(f.CompressedData.Available(), len(p))
		f.written += int64(maxWritable)
		f.CompressedData.Write(p[:maxWritable])
		p = p[maxWritable:]
	}

	if len(p) > 0 {
		if f.Overflow == nil {
			f.Overflow, err = os.CreateTemp("", "pzip-overflow")
			if err != nil {
				return len(p), errors.New("ERROR: could not create temp overflow directory")
			}
		}

		_, err := f.Overflow.Write(p)
		if err != nil {
			return len(p), errors.Errorf("ERROR: could not write to temp overflow directory for %s", f.Header.Name)
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
// The oveflowed contents are written to a temporary file.
func (f *File) Overflowed() bool {
	return f.Overflow != nil
}

func (f *File) setNameRelativeTo(root string) error {
	relativeToRoot, err := filepath.Rel(root, f.Path)
	if err != nil {
		return errors.Errorf("ERROR: could not find relative path of %s to root %s", f.Path, root)
	}
	f.Header.Name = filepath.Join(filepath.Base(root), relativeToRoot)
	return nil
}
