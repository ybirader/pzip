![logo-5](https://github.com/ybirader/pzip/assets/68111562/0b3cee2c-1af0-4753-b088-8a488f8ff642)

# pzip
pzip, short for parallel-zip, is a blazing fast concurrent zip archiver and extractor.

## Features

- Archives files and directories into a valid zip archive, using DEFLATE.
- Preserves modification times of files.
- Files are read and compressed concurrently

## Installation

### Command Line

For command-line usage, we provide two binaries which can be installed separately:
- pzip- concurrent zip archiving
- punzip- concurrent zip extraction

To install, run:

### macOS

For zip archiving: `brew install ybirader/pzip/pzip`
For zip extraction: `brew install ybirader/pzip/punzip`

#### Debian, Ubuntu, Raspbian

For the latest stable release:

```
curl -1sLf 'https://dl.cloudsmith.io/public/pzip/stable/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install pzip
```

```
curl -1sLf 'https://dl.cloudsmith.io/public/pzip/stable/setup.deb.sh' | sudo -E bash
sudo apt update
sudo apt install punzip
```

### Go

Alternatively, if you have Go installed:
```
go install github.com/ybirader/pzip
```

### Build from source

To build from source, we require Go 1.21 or newer.

1. Clone the repository by running `git clone "https://github.com/ybirader/pzip.git"`
2. Build both pzip and punzip by running `make build` or build separately via `cd cmd/pzip && go build` and `cd cmd/punzip && go build`

## Usage

### Archiving

`pzip`'s API is similar to that of the standard zip utlity found on most *-nix systems.

```
pzip /path/to/compressed.zip path/to/file_or_directory1 path/to/file_or_directory2 ... path/to/file_or_directoryN
```

Alternatively, pzip can be imported as a package

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
or by passing the `ArchiverConcurrency` option:
```go
archiver, err := pzip.NewArchiver(archive, ArchiverConcurrency(2))
```

### Extraction

`punzip`'s API is similar to that of the standard unzip utlity found on most *-nix systems.

```
punzip /path/to/compressed.zip
```

By default, `punzip` extracts into the current directory. We can extract to a particular path by:
```
punzip -d /path/to/output /path/to/compressed.zip
```

Using the Go package, we have:
```go
outputDirPath := "./output"
archivePath := "./archive.zip"

extractor, err := pzip.NewExtractor(outputDirPath)
if err != nil {
  log.Fatal(err)
}
defer extractor.Close()

err = extractor.Extract(context.Background(), archivePath)
if err != nil {
  log.Fatal(err)
}
```

As with pzip, we can configure the concurrency of the extractor using:

```
punzip --concurrency 2 /path/to/compressed.zip
```

Similarly, with the Go package, we pass in the `ExtractorConcurrency` option:
```go
extractor, err := pzip.NewExtractor(outputDirPath, ExtractorConcurrency(2))
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

## Acknowledgements

Many thanks to the folks at [Cloudsmith](https://cloudsmith.com) for graciously providing Debian package hosting. Cloudsmith is the only fully hosted, cloud-native, universal package management solution, that enables your organization to create, store and share packages in any format, to any place, with total confidence.

