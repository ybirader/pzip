package main_test

import (
	"path/filepath"
	"testing"

	"github.com/pzip/adapters/cli"
	"github.com/pzip/specifications"
)

const (
	testdataRoot = "../../testdata"
	archivePath  = testdataRoot + "/archive.zip"
	dirPath      = testdataRoot + "/hello"
)

func TestPzip(t *testing.T) {
	binPath, cleanup, err := cli.BuildBinary()
	if err != nil {
		t.Fatal("ERROR: could not build binary", err)
	}
	t.Cleanup(cleanup)
	t.Run("archives directory", func(t *testing.T) {
		if testing.Short() {
			t.Skip()
		}

		absArchivePath, err := filepath.Abs(archivePath)
		if err != nil {
			t.Fatalf("ERROR: could not get path to archive %s", archivePath)
		}

		absDirPath, err := filepath.Abs(dirPath)
		if err != nil {
			t.Fatalf("ERROR: could not get path to directory %s", dirPath)
		}

		driver := cli.NewDriver(binPath, absArchivePath, absDirPath)

		specifications.ArchiveDir(t, driver)
	})
}
