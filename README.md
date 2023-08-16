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
go get github.com/ybirader/pzip
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

dirPath := "./hello"

err = archiver.ArchiveDir(dirPath)
if err != nil {
  log.Fatal(err)
}

files := []string{"./hello.txt", "./bye.md"}
archiver.ArchiveFiles(files...)
```

Upcoming features:

- add context to gracefully stop archiving midway
- add flag to maintain unix file permissions i.e. mode of original file
- add support for symbolic links
- add flag to support skipping compression i.e. --skip-suffixes
- add ability to register different compressors



