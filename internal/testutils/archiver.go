package testutils

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func CreateTempArchive(t testing.TB, name string) (*os.File, func()) {
	t.Helper()

	archive, err := os.Create(name)
	assert.NoError(t, err, fmt.Sprintf("could not create archive %s: %v", name, err))

	cleanup := func() {
		archive.Close()
		os.RemoveAll(archive.Name())
	}

	return archive, cleanup
}

func GetFileInfo(t testing.TB, name string) fs.FileInfo {
	t.Helper()

	info, err := os.Stat(name)
	assert.NoError(t, err, fmt.Sprintf("could not get file info for %s", name))

	return info
}

func GetArchiveReader(t testing.TB, name string) *zip.ReadCloser {
	t.Helper()

	reader, err := zip.OpenReader(name)
	assert.NoError(t, err)

	return reader
}

func AssertArchiveContainsFile(t testing.TB, files []*zip.File, name string) {
	t.Helper()

	_, found := Find(files, func(f *zip.File) bool {
		return f.Name == name
	})

	if !found {
		t.Errorf("expected file %s to be in archive but wasn't", name)
	}
}

func Find[T any](elements []T, cb func(element T) bool) (T, bool) {
	for _, e := range elements {
		if cb(e) {
			return e, true
		}
	}

	return *new(T), false
}

func GetAllFiles(t testing.TB, dirPath string) []fs.FileInfo {
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

func GetOutput(t testing.TB, cmd *exec.Cmd) string {
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal("ERROR: could not get output of cmd", string(out), err)
	}

	return string(out)
}

func Map[T, K any](elements []T, cb func(element T) K) []K {
	results := make([]K, len(elements))

	for i, element := range elements {
		results[i] = cb(element)
	}

	return results
}
