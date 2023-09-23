package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"

	"github.com/pzip"
)

const description = "punzip is a tool for extracting files concurrently."

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
	}

	cli := pzip.ExtractorCLI{ArchivePath: args[0], DirPath: ".", Concurrency: concurrency}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop()
	}()

	err := cli.Extract(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
