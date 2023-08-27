package pzip_test

import (
	"os"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/pzip"
	"github.com/pzip/internal/testutils"
)

func TestCLI(t *testing.T) {
	t.Run("archives a directory", func(t *testing.T) {
		dirPath := "testdata/hello"
		archivePath := "testdata/archive.zip"
		defer os.RemoveAll(archivePath)

		cli := pzip.CLI{archivePath, dirPath}
		cli.Archive()

		archiveReader := testutils.GetArchiveReader(t, archivePath)
		defer archiveReader.Close()

		assert.Equal(t, 4, len(archiveReader.File))
	})
}
