package pzip

import (
	"io/fs"
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
	t.Run("writes decompressed archive files to output directory", func(t *testing.T) {
		err := os.Mkdir(outputDirPath, 0755)
		assert.NoError(t, err)
		defer os.RemoveAll(outputDirPath)

		extractor := NewExtractor(outputDirPath)
		err = extractor.Extract(testArchiveFixture)
		assert.NoError(t, err)

		files := getAllFiles(t, filepath.Join(outputDirPath, "hello"))
		assert.Equal(t, []string{"hello.txt", "nested", "hello.md"}, Map(files, func(element fs.FileInfo) string {
			return element.Name()
		}))

		helloFileInfo := files[0]
		assert.NotZero(t, helloFileInfo.Size())
	})
}

func getAllFiles(t testing.TB, dirPath string) []fs.FileInfo {
	var result []fs.FileInfo

	err := filepath.Walk(dirPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if dirPath == path {
			return nil
		}

		result = append(result, info)

		return nil
	})

	if err != nil {
		t.Fatalf("could not walk directory %s: %v", dirPath, err)
	}

	return result
}

func Map[T, K any](elements []T, cb func(element T) K) []K {
	results := make([]K, len(elements))

	for i, element := range elements {
		results[i] = cb(element)
	}

	return results
}
