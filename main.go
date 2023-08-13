package main

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/flate"
)

type Archiver struct {
	Dest *os.File
	w    *zip.Writer
}

type File struct {
	Path string
	Info fs.FileInfo
}

func NewArchiver(archive *os.File) *Archiver {
	return &Archiver{Dest: archive, w: zip.NewWriter(archive)}
}

func (a *Archiver) ArchiveDir(root string) error {
	err := a.walkDir(root)

	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) walkDir(root string) error {
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		f := File{Path: path, Info: info}

		err = a.archive(&f)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) ArchiveFiles(files ...string) error {
	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		f := File{Path: path, Info: info}
		err = a.archive(&f)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Archiver) Close() error {
	err := a.w.Close()
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) archive(f *File) error {
	err := a.writeFile(f)

	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) writeFile(f *File) error {
	writer, err := a.createFile(f.Info)
	if err != nil {
		return err
	}

	if f.Info.IsDir() {
		return nil
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}

	err = a.writeContents(writer, file)
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) createFile(info fs.FileInfo) (io.Writer, error) {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return nil, err
	}

	writer, err := a.w.CreateHeader(header)
	if err != nil {
		return nil, err
	}

	return writer, nil
}

func (a *Archiver) writeContents(w io.Writer, r io.Reader) error {
	_, err := io.Copy(w, r)
	if err != nil {
		return err
	}

	return nil
}

const DefaultCompression = -1

func compressToBuffer(buf *bytes.Buffer, path string) {
	file, _ := os.Open(path)
	defer file.Close()

	compressor, _ := flate.NewWriter(buf, DefaultCompression)
	io.Copy(compressor, file)
	compressor.Close()
}

func main() {
}
