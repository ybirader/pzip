package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pzip"
)

const description = "pzip is a tool for archiving files concurrently"

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

	cli := pzip.CLI{ArchivePath: archivePath, Files: os.Args[2:]}
	err := cli.Archive()

	if err != nil {
		log.Fatal(err)
	}
}
