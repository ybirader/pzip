package main

import (
	"archive/zip"
	"io"
	"os"
)

type Archiver struct {
	Dest *os.File
	w    *zip.Writer
}

func NewArchiver(archive *os.File) *Archiver {
	return &Archiver{Dest: archive, w: zip.NewWriter(archive)}
}

func (a *Archiver) Archive(r io.Reader) error {
	writer, err := a.w.Create("hello.txt")
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, r)
	if err != nil {
		return err
	}

	a.w.Close()
	return nil
}

func main() {

}
