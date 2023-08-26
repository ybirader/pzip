package pzip

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	filebuffer "github.com/pzip/file_buffer"
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

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		archiver.ArchiveFiles(helloTxtFileFixture)
		archiver.Close()

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

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		archiver.ArchiveFiles(helloTxtFileFixture)
		archiver.Close()

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

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		archiver.ArchiveFiles(helloTxtFileFixture, helloMarkdownFileFixture)
		archiver.Close()

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 2, len(archiveReader.File))
	})

	t.Run("archives a directory of files", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		err = archiver.ArchiveDir(helloDirectoryFixture)
		assert.NoError(t, err)
		archiver.Close()

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 3, len(archiveReader.File))
	})

	t.Run("can archive files separately", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		err = archiver.ArchiveFiles(helloTxtFileFixture)
		assert.NoError(t, err)
		err = archiver.ArchiveFiles(helloMarkdownFileFixture)
		assert.NoError(t, err)
		archiver.Close()

		archiveReader := getArchiveReader(t, archive.Name())
		defer archiveReader.Close()

		assert.Equal(t, 2, len(archiveReader.File))
	})
}

func TestCompressToBuffer(t *testing.T) {
	t.Run("compresses file to buffer using default deflate compression", func(t *testing.T) {
		archive, cleanup := createTempArchive(t, archivePath)
		defer cleanup()

		archiver, err := NewArchiver(archive)
		assert.NoError(t, err)
		info := getFileInfo(t, helloTxtFileFixture)
		file := filebuffer.File{Path: helloTxtFileFixture, Info: info}

		buf := bytes.Buffer{}
		archiver.compressToBuffer(&buf, &file)

		want := []byte{0, 14, 0, 241, 255, 104, 101, 108, 108, 111, 44, 32, 119, 111, 114, 108, 100, 33, 10, 3, 0}

		assert.Equal(t, want, buf.Bytes())
	})
}

func TestFileWriter(t *testing.T) {
	t.Run("writes correct header", func(t *testing.T) {
		t.Run("with file name relative to archive root when file path is absolute", func(t *testing.T) {
			archive, cleanup := createTempArchive(t, archivePath)
			defer cleanup()

			archiver, err := NewArchiver(archive)
			assert.NoError(t, err)

			info := getFileInfo(t, helloTxtFileFixture)

			absPath, err := filepath.Abs(helloTxtFileFixture)
			assert.NoError(t, err)
			file := filebuffer.File{Path: absPath, Info: info}

			archiver.createHeader(&file)

			assert.Equal(t, "hello.txt", file.Header.Name)
		})

		t.Run("with file name relative to archive root when file path is relative", func(t *testing.T) {
			archive, cleanup := createTempArchive(t, archivePath)
			defer cleanup()

			archiver, err := NewArchiver(archive)
			assert.NoError(t, err)

			info := getFileInfo(t, helloTxtFileFixture)
			file := filebuffer.File{Path: helloTxtFileFixture, Info: info}

			archiver.createHeader(&file)

			assert.Equal(t, "hello.txt", file.Header.Name)
		})

		t.Run("with file names relative to archive root for directories", func(t *testing.T) {
			archive, cleanup := createTempArchive(t, archivePath)
			defer cleanup()

			archiver, err := NewArchiver(archive)
			assert.NoError(t, err)

			archiver.changeRoot(helloDirectoryFixture)
			filePath := "nested/hello.md"

			info := getFileInfo(t, filepath.Join(archiver.chroot, filePath))
			file := filebuffer.File{Path: filePath, Info: info}

			archiver.createHeader(&file)

			assert.Equal(t, "nested/hello.md", file.Header.Name)
		})

		t.Run("with deflate method and correct mod time, mode, data descriptor and extended timestamp for files", func(t *testing.T) {
			archive, cleanup := createTempArchive(t, archivePath)
			defer cleanup()

			archiver, err := NewArchiver(archive)
			assert.NoError(t, err)

			info := getFileInfo(t, helloTxtFileFixture)
			file := filebuffer.File{Path: helloTxtFileFixture, Info: info}
			archiver.compress(&file)

			archiver.createHeader(&file)

			assert.Equal(t, zip.Deflate, file.Header.Method)
			assertMatchingTimes(t, info.ModTime(), file.Header.Modified)
			assert.Equal(t, info.Mode(), file.Header.Mode())

			assert.NotZero(t, file.Header.CRC32)
			assert.Equal(t, uint64(file.CompressedData.Len()), file.Header.CompressedSize64)
			assert.Equal(t, uint64(info.Size()), file.Header.UncompressedSize64)

			assertExtendedTimestamp(t, file.Header)
		})

		t.Run("with no data descriptor directories", func(t *testing.T) {
			archive, cleanup := createTempArchive(t, archivePath)
			defer cleanup()

			archiver, err := NewArchiver(archive)
			assert.NoError(t, err)

			info := getFileInfo(t, filepath.Join(helloDirectoryFixture, "/nested"))
			file := filebuffer.File{Path: filepath.Join(helloDirectoryFixture, "/nested"), Info: info}

			archiver.createHeader(&file)

			assert.Equal(t, zip.Store, file.Header.Method)
			assert.Zero(t, file.Header.CRC32)
			assert.Equal(t, 0, file.Header.CompressedSize64)
			assert.Equal(t, 0, file.Header.UncompressedSize64)
		})
	})
}

func assertExtendedTimestamp(t testing.TB, hdr *zip.FileHeader) {
	want := make([]byte, 2)
	binary.LittleEndian.PutUint16(want, extendedTimestampTag)
	got := hdr.Extra[:2]
	assert.Equal(t, want, got, "expected header to contain extended timestamp")
}

func BenchmarkArchive(b *testing.B) {
	archive, cleanup := createTempArchive(b, archivePath)
	defer cleanup()

	archiver, err := NewArchiver(archive)
	assert.NoError(b, err)
	defer archiver.Close()
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
