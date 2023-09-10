package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/pzip"
)

const description = "pzip is a tool for archiving files concurrently."

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, description)
		fmt.Fprintln(os.Stderr, "\nUsage:")
		flag.PrintDefaults()
	}

	var concurrency int
	flag.IntVar(&concurrency, "concurrency", runtime.GOMAXPROCS(0), "allow up to n compression routines")

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		return
	} else if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "pzip error: invalid usage")
		return
	}

	cli := pzip.CLI{ArchivePath: args[0], Files: args[1:], Concurrency: concurrency}
	err := cli.Archive()

	if err != nil {
		log.Fatal(err)
	}
}
