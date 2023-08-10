package main

import (
	"archive/zip"
	"fmt"
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

func (a *Archiver) Archive(r io.Reader) {
	writer, _ := a.w.Create("hello.txt")
	fmt.Fprint(writer, r)
	a.w.Close()
}

func main() {

}
