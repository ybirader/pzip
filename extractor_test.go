package pzip

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/pzip/internal/testutils"
)

const (
	testArchiveFixture = testdataRoot + "test.zip" // test.zip fixture is an archive of the helloDirectory fixture
	outputDirPath      = testdataRoot + "test"
)

func TestExtract(t *testing.T) {
	t.Run("writes decompressed archive files to output directory", func(t *testing.T) {
		err := os.Mkdir(outputDirPath, 0755)
		assert.NoError(t, err)
		defer os.RemoveAll(outputDirPath)

		extractor := NewExtractor(outputDirPath)
		defer extractor.Close()
		err = extractor.Extract(testArchiveFixture)
		assert.NoError(t, err)

		files := testutils.GetAllFiles(t, filepath.Join(outputDirPath, "hello"))
		assert.Equal(t, []string{"hello.txt", "nested", "hello.md"}, testutils.Map(files, func(element fs.FileInfo) string {
			return element.Name()
		}))

		helloFileInfo := files[0]
		assert.NotZero(t, helloFileInfo.Size())
	})
}
