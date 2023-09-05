package pool

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"path/filepath"

	"github.com/pkg/errors"
)

type File struct {
	Path           string
	Info           fs.FileInfo
	CompressedData bytes.Buffer
	Header         *zip.FileHeader
	Status         Status
}

type Status int

const (
	defaultBufferSize        = 1000000
	FileFinished      Status = iota
	FileFull
)

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
		return f.CompressedData.Write(p[:int(maxWritable)])
	}

	f.Status = FileFull
	return len(p), nil
}

func (f *File) setNameRelativeTo(root string) error {
	relativeToRoot, err := filepath.Rel(root, f.Path)
	if err != nil {
		return errors.Errorf("ERROR: could not find relative path of %s to root %s", f.Path, root)
	}
	f.Header.Name = filepath.Join(filepath.Base(root), relativeToRoot)
	return nil
}
