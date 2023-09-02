package pool

import (
	"archive/zip"
	"bytes"
	"io/fs"

	"github.com/pkg/errors"
)

type File struct {
	Path           string
	Info           fs.FileInfo
	CompressedData bytes.Buffer
	Header         *zip.FileHeader
}

func NewFile(path string, info fs.FileInfo) (File, error) {
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return File{}, errors.Errorf("ERROR: could not get file info header for %s: %v", path, err)
	}

	return File{Path: path, Info: info, Header: hdr}, nil
}
