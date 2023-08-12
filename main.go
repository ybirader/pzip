package main

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Archiver struct {
	Dest *os.File
	w    *zip.Writer
}

func NewArchiver(archive *os.File) *Archiver {
	return &Archiver{Dest: archive, w: zip.NewWriter(archive)}
}

func (a *Archiver) ArchiveDir(root string) error {
	files := make(map[string]fs.FileInfo, 0)

	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		files[path] = info

		return nil
	})

	if err != nil {
		return err
	}

	err = a.archive(files)
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) ArchiveFiles(files ...string) error {
	f := make(map[string]fs.FileInfo, 0)

	for _, name := range files {
		info, err := os.Lstat(name)
		if err != nil {
			return err
		}

		f[name] = info
	}

	err := a.archive(f)
	if err != nil {
		return err
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

func (a *Archiver) archive(files map[string]fs.FileInfo) error {
	for path, info := range files {
		err := a.WriteFile(path, info)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Archiver) WriteFile(path string, info fs.FileInfo) error {
	writer, err := a.createFile(info)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	file, err := os.Open(path)
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

func main() {
}
