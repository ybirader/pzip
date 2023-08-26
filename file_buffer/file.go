package filebuffer

import (
	"archive/zip"
	"bytes"
	"io/fs"
)

type File struct {
	Path           string
	Info           fs.FileInfo
	CompressedData bytes.Buffer
	Header         *zip.FileHeader
}
