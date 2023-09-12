# pzip
pzip, short for parallel-zip, is a blazing fast concurrent zip archiver.

## Features

- Archives files and directories into a valid zip archive, using DEFLATE
- Preserves modification times of files.
- Files are read and compressed concurrently

### Installation

To install pzip, run `brew install pzip` [TODO: Add brew package]

You can also use pzip as a library by importing the go package:
```
go install github.com/ybirader/pzip
```

### Usage

pzip's API has been designed to mimic the standard zip utlity found on most *-nix systems.

```
pzip /path/to/compressed.zip path/to/file_or_directory
```

Alternatively, pzip can be imported as a library

```go
archive, err := os.Create("archive.zip")
if err != nil {
  log.Fatal(err)
}

archiver, err := pzip.NewArchiver(archive)
if err != nil {
  log.Fatal(err)
}
defer archiver.Close()

files := []string{ "./hello", "./hello.txt", "./bye.md" }

err = archiver.Archive(context.Background(), files)
if err != nil {
  log.Fatal(err)
}
```


### Benchmarks

We use Matt Mahoney's [sample directory](https://mattmahoney.net/dc/10gb.html) in our benchmark

Using the standard `zip` utlity found on most *nix systems, we get the following time to archive:
```
real    14m31.809s
user    13m12.833s
sys     0m24.193s
```

The size of the resulting archive is 4.51 GB

Running the same benchmark with pzip, we find that:

```
goos: darwin
goarch: amd64
pkg: github.com/pzip/cmd/cli
cpu: Intel(R) Core(TM) i5-8259U CPU @ 2.30GHz
BenchmarkPzip-8                1        81600764936 ns/op           7928 B/op         32 allocs/op
PASS
ok      github.com/pzip/cmd/cli 83.847s
```

The size of the resulting zip was slightly larger at: 4.62 GB.

Overall, this is over 10x faster! And this is with no optimizations for memory etc.

Upcoming features:

- add flag to maintain unix file permissions i.e. mode of original file
- add support for symbolic links
- add flag to support skipping compression i.e. --skip-suffixes
- add ability to register different compressors

## License

pzip is released under the [MIT License](https://opensource.org/license/mit/).

