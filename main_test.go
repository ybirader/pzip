package main

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

const (
	testdataRoot             = "testdata/"
	archivePath              = testdataRoot + "archive.zip"
	helloTxtFileFixture      = testdataRoot + "hello.txt"
	helloMarkdownFileFixture = testdataRoot + "hello.md"
	helloDirectoryFixture    = testdataRoot + "hello/"
)

func TestArchive(t *testing.T) {
	t.Run("archives a single file with a name", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver := NewArchiver(archive)
		archiver.ArchiveFiles(helloTxtFileFixture)

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 1, len(archiveReader.File))
		assertArchiveContainsFile(t, archiveReader.File, "hello.txt")

		info := getFileInfo(t, helloTxtFileFixture)

		got := archiveReader.File[0].UncompressedSize64
		want := uint64(info.Size())

		assert.Equal(t, want, got, "expected file %s to have raw size %d but got %d", info.Name(), want, got)
	})

	t.Run("retains the last modified date of an archived file", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver := NewArchiver(archive)
		archiver.ArchiveFiles(helloTxtFileFixture)

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		info := getFileInfo(t, helloTxtFileFixture)

		archivedFile, found := Find(archiveReader.File, func(file *zip.File) bool {
			return file.Name == "hello.txt"
		})
		assert.True(t, found)

		assertMatchingTimes(t, archivedFile.Modified, info.ModTime())
	})

	t.Run("archives two files", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver := NewArchiver(archive)
		archiver.ArchiveFiles(helloTxtFileFixture, helloMarkdownFileFixture)

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 2, len(archiveReader.File))
	})

	t.Run("archives a directory of files", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver := NewArchiver(archive)
		archiver.ArchiveDir(helloDirectoryFixture)

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 3, len(archiveReader.File))
	})
}

func BenchmarkArchive(b *testing.B) {
	archive, cleanup := createTempArchive(b, archivePath)
	defer cleanup()

	archiver := NewArchiver(archive)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		archiver.ArchiveFiles(helloTxtFileFixture, helloMarkdownFileFixture)
	}
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

func assertArchiveContainsFile(t testing.TB, files []*zip.File, name string) {
	t.Helper()

	_, found := Find(files, func(f *zip.File) bool {
		return f.Name == name
	})

	if !found {
		t.Errorf("expected file %s to be in archive but wasn't", name)
	}
}

func assertMatchingTimes(t testing.TB, t1, t2 time.Time) {
	t.Helper()

	assert.True(t,
		t1.Year() == t2.Year() && t1.YearDay() == t2.YearDay() && t1.Second() == t2.Second(),
		fmt.Sprintf("expected %+v to match %+v but didn't", t1, t2))
}

func getFileInfo(t testing.TB, name string) fs.FileInfo {
	t.Helper()

	info, err := os.Stat(name)
	assert.NoError(t, err, fmt.Sprintf("could not get file into fot %s", name))

	return info
}

func Find[T any](elements []T, cb func(element T) bool) (T, bool) {
	for _, e := range elements {
		if cb(e) {
			return e, true
		}
	}

	return *new(T), false
}
