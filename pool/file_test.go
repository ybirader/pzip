package pool_test

import (
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/pzip/internal/testutils"
	"github.com/pzip/pool"
)

const (
	testdataRoot             = "../testdata/"
	archivePath              = testdataRoot + "archive.zip"
	helloTxtFileFixture      = testdataRoot + "hello.txt"
	helloMarkdownFileFixture = testdataRoot + "hello.md"
	helloDirectoryFixture    = testdataRoot + "hello/"
)

func TestNewFile(t *testing.T) {
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
}
