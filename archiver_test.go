package pzip

import (
	"archive/zip"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/pzip/internal/testutils"
	"github.com/pzip/pool"
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
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		archiver.Archive([]string{helloTxtFileFixture})
		archiver.Close()

		archiveReader := testutils.GetArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 1, len(archiveReader.File))
		testutils.AssertArchiveContainsFile(t, archiveReader.File, "hello.txt")

		info := testutils.GetFileInfo(t, helloTxtFileFixture)

		got := archiveReader.File[0].UncompressedSize64
		want := uint64(info.Size())

		assert.Equal(t, want, got, "expected file %s to have raw size %d but got %d", info.Name(), want, got)
	})

	t.Run("retains the last modified date of an archived file", func(t *testing.T) {
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		archiver.Archive([]string{helloTxtFileFixture})
		archiver.Close()

		archiveReader := testutils.GetArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		info := testutils.GetFileInfo(t, helloTxtFileFixture)

		archivedFile, found := testutils.Find(archiveReader.File, func(file *zip.File) bool {
			return file.Name == "hello.txt"
		})
		assert.True(t, found)

		assertMatchingTimes(t, archivedFile.Modified, info.ModTime())
	})

	t.Run("archives two files", func(t *testing.T) {
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		archiver.Archive([]string{helloTxtFileFixture, helloMarkdownFileFixture})
		archiver.Close()

		archiveReader := testutils.GetArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 2, len(archiveReader.File))
	})

	t.Run("archives a directory of files", func(t *testing.T) {
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		err = archiver.Archive([]string{helloDirectoryFixture})
		assert.NoError(t, err)
		archiver.Close()

		archiveReader := testutils.GetArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 4, len(archiveReader.File))
	})

	t.Run("can archive files separately", func(t *testing.T) {
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		err = archiver.Archive([]string{helloTxtFileFixture})
		assert.NoError(t, err)
		err = archiver.Archive([]string{helloMarkdownFileFixture})
		assert.NoError(t, err)
		archiver.Close()

		archiveReader := testutils.GetArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 2, len(archiveReader.File))
	})
}

func TestCompress(t *testing.T) {
	t.Run("compresses file to buffer using default deflate compression", func(t *testing.T) {
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)

		info := testutils.GetFileInfo(t, helloTxtFileFixture)
		file, err := pool.NewFile(helloTxtFileFixture, info)
		assert.NoError(t, err)

		archiver.compress(&file)

		want := []byte{0, 14, 0, 241, 255, 104, 101, 108, 108, 111, 44, 32, 119, 111, 114, 108, 100, 33, 10, 3, 0}

		assert.Equal(t, want, file.CompressedData.Bytes())
	})
}

func TestPopulateHeader(t *testing.T) {
	t.Run("with file name relative to archive root when file path is relative", func(t *testing.T) {
		info := testutils.GetFileInfo(t, helloTxtFileFixture)
		file, err := pool.NewFile(helloTxtFileFixture, info)
		assert.NoError(t, err)

		assert.Equal(t, "hello.txt", file.Header.Name)
	})

	t.Run("with file name relative to archive root when file path is absolute", func(t *testing.T) {
		absFilePath, err := filepath.Abs(helloTxtFileFixture)
		assert.NoError(t, err)
		info := testutils.GetFileInfo(t, absFilePath)
		file, err := pool.NewFile(absFilePath, info)
		assert.NoError(t, err)

		assert.Equal(t, "hello.txt", file.Header.Name)
	})

	t.Run("with file name relative to archive root for directories", func(t *testing.T) {
		filePath := filepath.Join(helloDirectoryFixture, "nested/hello.md")
		info := testutils.GetFileInfo(t, filePath)

		file, err := pool.NewFile(filePath, info)
		assert.NoError(t, err)

		err = file.SetNameRelativeTo(helloDirectoryFixture)
		assert.NoError(t, err)

		assert.Equal(t, "hello/nested/hello.md", file.Header.Name)
	})
	t.Run("with deflate method and correct mod time, mode, data descriptor and extended timestamp for files", func(t *testing.T) {
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)

		info := testutils.GetFileInfo(t, helloTxtFileFixture)
		file, err := pool.NewFile(helloTxtFileFixture, info)
		assert.NoError(t, err)

		archiver.compress(&file)
		archiver.populateHeader(&file)

		assert.Equal(t, zip.Deflate, file.Header.Method)
		assertMatchingTimes(t, info.ModTime(), file.Header.Modified)
		assert.Equal(t, info.Mode(), file.Header.Mode())
		assert.NotZero(t, file.Header.CRC32)
		assert.Equal(t, uint64(file.CompressedData.Len()), file.Header.CompressedSize64)
		assert.Equal(t, uint64(info.Size()), file.Header.UncompressedSize64)
		assertExtendedTimestamp(t, file.Header.Extra)
	})

	t.Run("for directories", func(t *testing.T) {
		archive, cleanup := testutils.CreateTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)

		filePath := filepath.Join(helloDirectoryFixture, "nested")
		info := testutils.GetFileInfo(t, filePath)
		file, err := pool.NewFile(filePath, info)
		assert.NoError(t, err)
		file.SetNameRelativeTo(helloDirectoryFixture)

		archiver.compress(&file)
		archiver.populateHeader(&file)

		assert.Equal(t, "hello/nested/", file.Header.Name)
		assert.Equal(t, zip.Store, file.Header.Method)
		assert.Zero(t, file.Header.CRC32)
		assert.Equal(t, 0, file.Header.CompressedSize64)
		assert.Equal(t, 0, file.Header.UncompressedSize64)
	})
}

func assertExtendedTimestamp(t testing.TB, extraField []byte) {
	want := make([]byte, 2)
	binary.LittleEndian.PutUint16(want, extendedTimestampTag)
	got := extraField[:2]
	assert.Equal(t, want, got, "expected header to contain extended timestamp")
}

func assertMatchingTimes(t testing.TB, t1, t2 time.Time) {
	t.Helper()

	assert.True(t,
		t1.Year() == t2.Year() && t1.YearDay() == t2.YearDay() && t1.Second() == t2.Second(),
		fmt.Sprintf("expected %+v to match %+v but didn't", t1, t2))
}

func BenchmarkArchive(b *testing.B) {
	archive, cleanup := testutils.CreateTempArchive(b, archivePath)
	defer cleanup()

	archiver, err := NewArchiver(archive)
	assert.NoError(b, err)
	defer archiver.Close()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		archiver.Archive([]string{helloTxtFileFixture, helloMarkdownFileFixture})
	}
}
