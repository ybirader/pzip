package main

import (
	"log"
	"os"

	"github.com/pzip"
)

func main() {
	archivePath := os.Args[1]

	cli := pzip.CLI{ArchivePath: archivePath, Files: os.Args[2:]}
	err := cli.Archive()

	if err != nil {
		log.Fatal(err)
	}
}
