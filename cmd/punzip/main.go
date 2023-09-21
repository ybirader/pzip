package main

import (
	"os"

	"github.com/pzip"
)

func main() {
	archivePath := os.Args[1]

	cli := pzip.ExtractorCLI{ArchivePath: archivePath, DirPath: "."}
	cli.Extract()
}
