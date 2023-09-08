package pool

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

const DefaultBufferSize = 2 * 1024 * 1024

var FilePool = sync.Pool{
	New: func() any {
		return &File{CompressedData: bytes.NewBuffer(make([]byte, DefaultBufferSize))}
	},
}

type File struct {
	Path           string
	Info           fs.FileInfo
	Header         *zip.FileHeader
	CompressedData *bytes.Buffer
	Overflow       *os.File
	written        int64
}

func NewFile(path string, info fs.FileInfo, relativeTo string) (*File, error) {
	f := FilePool.Get().(*File)
	err := f.Reset(path, info, relativeTo)
	return f, err
}

func (f *File) Reset(path string, info fs.FileInfo, relativeTo string) error {
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		errors.Errorf("ERROR: could not get file info header for %s: %v", path, err)
	}
	f.Path = path
	f.Info = info
	f.Header = hdr
	f.CompressedData.Reset()
	f.Overflow = nil
	f.written = 0

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
