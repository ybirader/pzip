package main

import (
	"archive/zip"
	"os"
)

type Archiver struct {
	Dest *os.File
	w    *zip.Writer
}

func NewArchiver(archive *os.File) *Archiver {
	return &Archiver{Dest: archive, w: zip.NewWriter(archive)}
}

func (a *Archiver) Archive() {
	a.w.Create("hello.txt")
	a.w.Close()
}

func main() {

}
