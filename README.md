![logo-5](https://github.com/ybirader/pzip/assets/68111562/0b3cee2c-1af0-4753-b088-8a488f8ff642)

# pzip
pzip, short for parallel-zip, is a blazing fast concurrent zip archiver.

## Features

- Archives files and directories into a valid zip archive, using DEFLATE.
- Preserves modification times of files.
- Files are read and compressed concurrently

## Installation

To install pzip, run:

### macOS

 `brew install pzip/tap/pzip`

### Debian, Ubuntu, Raspbian

```
sudo apt update
sudo apt install pzip
```

### Go

Alternatively, if you have Go installed:
```
go install github.com/ybirader/pzip
```

### Build from source

To build from source, we require Go 1.21 or newer.

1. Clone the repository by running `git clone "https://github.com/ybirader/pzip.git"`
2. Build by running `make build` or `cd cmd/cli && go build`

## Usage

pzip's API is similar to that of the standard zip utlity found on most *-nix systems.

```
pzip /path/to/compressed.zip path/to/file_or_directory1 path/to/file_or_directory2 ... path/to/file_or_directoryN
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

The concurrency of the archiver can be configured using the corresponding flag:
```
pzip --concurrency 2 /path/to/compressed.zip path/to/file_or_directory1 path/to/file_or_directory2 ... path/to/file_or_directoryN

```
or by using passing the `Concurrency` option:
```go
archiver, err := pzip.NewArchiver(archive, Concurrency(2))
```

### Benchmarks

pzip was benchmarked using Matt Mahoney's [sample directory](https://mattmahoney.net/dc/10gb.html).

Using the standard `zip` utlity, we get the following time to archive:
```
real    14m31.809s
user    13m12.833s
sys     0m24.193s
```

Running the same benchmark with pzip, we find that:

```
real    0m56.851s
user    3m32.619s
sys     1m25.040s
```

## Contributing

To contribute to pzip, first submit or comment in an issue to discuss your contribution, then open a pull request (PR).

## License

pzip is released under the [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0) license.

