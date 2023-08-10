package main

import (
	"archive/zip"
	"fmt"
	"os"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestArchive(t *testing.T) {
	t.Run("archives a single empty file called hello.txt", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, "testdata/archive.zip")
		defer cleanup()

		writer := zip.NewWriter(archive)
		archiver := Archiver{Dest: archive, W: writer}
		archiver.Archive()

		archiveReader, err := zip.OpenReader(archive.Name())
		assert.NoError(t, err)
		defer archiveReader.Close()

		assert.Equal(t, 1, len(archiveReader.File))
		assert.Equal(t, "hello.txt", archiveReader.File[0].Name)
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
