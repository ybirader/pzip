package pool

import (
	"archive/zip"
	"bytes"
	"io/fs"
)

type File struct {
	Name           string
	Path           string
	Info           fs.FileInfo
	CompressedData bytes.Buffer
	Header         *zip.FileHeader
}
