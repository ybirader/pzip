package pool

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const defaultBufferSize = 2 * 1024 * 1024

type Overflow interface {
	io.ReadWriteSeeker
	io.Closer
	Name() string
}

type File struct {
	Path           string
	Info           fs.FileInfo
	CompressedData bytes.Buffer
	Header         *zip.FileHeader
	Overflow       Overflow
	written        int64
}

func NewFile(path string, info fs.FileInfo, relativeTo string) (File, error) {
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return File{}, errors.Errorf("ERROR: could not get file info header for %s: %v", path, err)
	}

	f := File{Path: path, Info: info, Header: hdr, CompressedData: *bytes.NewBuffer(make([]byte, 0, defaultBufferSize))}
	if relativeTo != "" {
		f.setNameRelativeTo(relativeTo)
	}

	return f, nil
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
				return len(p), err
			}
		}

		_, err := f.Overflow.Write(p)
		if err != nil {
			return len(p), err
		}
		f.written += int64(len(p))
	}

	return len(p), nil
}

func (f *File) Written() int64 {
	return f.written
}

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
