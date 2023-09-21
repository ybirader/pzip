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
	t.Run("writes empty archive files to output directory", func(t *testing.T) {
		err := os.Mkdir(outputDirPath, 0755)
		assert.NoError(t, err)
		defer os.RemoveAll(outputDirPath)

		extractor := NewExtractor(outputDirPath)
		err = extractor.Extract(testArchiveFixture)
		assert.NoError(t, err)

		files, err := os.ReadDir(filepath.Join(outputDirPath, "hello"))
		assert.NoError(t, err)
		assert.Equal(t, 2, len(files))
		assertDirContains(t, files, "hello.txt")
		assertDirContains(t, files, "hello.txt")

		files, err = os.ReadDir(filepath.Join(outputDirPath, "hello", "nested"))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(files))
		assertDirContains(t, files, "hello.md")
	})
}

func assertDirContains(t testing.TB, files []fs.DirEntry, name string) {
	t.Helper()

	_, found := testutils.Find(files, func(element fs.DirEntry) bool {
		return element.Name() == name
	})

	if !found {
		t.Errorf("expected %+v to contain %s but didn't", files, name)
	}
}
