package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/alecthomas/assert/v2"
)

const (
	archivePath = "testdata/archive.zip"
)

func TestArchive(t *testing.T) {
	t.Run("archives a single empty file called hello.txt", func(t *testing.T) {
		file := bytes.NewBufferString("")
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver := NewArchiver(archive)
		archiver.Archive(file)

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 1, len(archiveReader.File))
		assert.Equal(t, "hello.txt", archiveReader.File[0].Name)
	})

	t.Run("archives a single non-empty file called hello.txt", func(t *testing.T) {
		file := bytes.NewBufferString("hello, world!")
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver := NewArchiver(archive)
		archiver.Archive(file)

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		got := archiveReader.File[0].UncompressedSize64
		want := uint64(file.Len())

		assert.Equal(t, want, got, "expected %s to have size %d but got %d", "hello.txt", want, got)
	})
}

func createTempArchive(t testing.TB, name string) (*os.File, func()) {
	t.Helper()

	archive, err := os.Create(name)
	assert.NoError(t, err, fmt.Sprintf("could not create archive %s: %v", name, err))

	cleanup := func() {
		archive.Close()
		os.RemoveAll(archive.Name())
	}

	return archive, cleanup
}

func getArchiveReader(t testing.TB, name string) *zip.ReadCloser {
	t.Helper()

	reader, err := zip.OpenReader(name)
	assert.NoError(t, err)

	return reader
}
