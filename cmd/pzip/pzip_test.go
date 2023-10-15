package main_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/ybirader/pzip/adapters/cli"
	"github.com/ybirader/pzip/internal/testutils"
	"github.com/ybirader/pzip/specifications"
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

	t.Run("outputs usage to stderr when no arguments or flags provided", func(t *testing.T) {
		pzip := exec.Command(binPath)
		out := testutils.GetOutput(t, pzip)

		assert.Contains(t, out, "pzip is a tool for archiving files concurrently.\n")
		assert.Contains(t, out, "Usage")
	})

	t.Run("outputs error when only one argument passed", func(t *testing.T) {
		pzip := exec.Command(binPath, "archive.zip")
		out := testutils.GetOutput(t, pzip)

		assert.Contains(t, out, "pzip error: invalid usage\n")
	})

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

		specifications.Archive(t, driver)
	})
}
