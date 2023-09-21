package pzip

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

const (
	testArchiveFixture = testdataRoot + "test.zip" // test.zip fixture is an archive of the helloDirectory fixture
	outputDirPath      = testdataRoot + "test"
)

func TestExtract(t *testing.T) {
	t.Run("writes one file to output directory", func(t *testing.T) {
		err := os.Mkdir(outputDirPath, 0755)
		assert.NoError(t, err)
		defer os.RemoveAll(outputDirPath)

		extractor := NewExtractor(outputDirPath)
		err = extractor.Extract(testArchiveFixture)
		assert.NoError(t, err)

		files, err := os.ReadDir(filepath.Join(outputDirPath, "hello"))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(files))
	})
}
