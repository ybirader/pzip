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
	Status         string
}

func NewFile(path string, info fs.FileInfo) (File, error) {
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return File{}, errors.Errorf("ERROR: could not get file info header for %s: %v", path, err)
	}

	return File{Path: path, Info: info, Header: hdr}, nil
}

func (f *File) SetNameRelativeTo(root string) error {
	relativeToRoot, err := filepath.Rel(root, f.Path)
	if err != nil {
		return errors.Errorf("ERROR: could not find relative path of %s to root %s", f.Path, root)
	}
	f.Header.Name = filepath.Join(filepath.Base(root), relativeToRoot)
	return nil
}
