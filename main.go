package main

import (
	"archive/zip"
	"os"
)

type Archiver struct {
	Dest *os.File
	W    *zip.Writer
}

func (a *Archiver) Archive() {
	a.W.Create("hello.txt")
	a.W.Close()
}

func main() {

}
