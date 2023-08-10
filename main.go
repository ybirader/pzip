package main

import (
	"archive/zip"
	"io"
	"os"
)

func Archive(archive *os.File, r io.Reader) {
	writer := zip.NewWriter(archive)
	defer writer.Close()

	writer.Create("hello.txt")
}

func main() {

}
