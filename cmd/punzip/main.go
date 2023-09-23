package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pzip"
)

const description = "punzip is a tool for extracting files concurrently."

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, description)
		fmt.Fprintln(os.Stderr, "\nUsage:")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		return
	}

	archivePath := os.Args[1]

	cli := pzip.ExtractorCLI{ArchivePath: archivePath, DirPath: "."}
	cli.Extract()
}
