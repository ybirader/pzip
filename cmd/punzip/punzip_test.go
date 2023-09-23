package main_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/pzip/adapters/cli"
	"github.com/pzip/internal/testutils"
	"github.com/pzip/specifications"
)

const (
	testdataRoot = "../../testdata"
	archivePath  = testdataRoot + "/test.zip"
)

func TestPunzip(t *testing.T) {
	binPath, cleanup, err := cli.BuildBinary()
	if err != nil {
		t.Fatal("ERROR: could not build binary", err)
	}
	t.Cleanup(cleanup)

	t.Run("outputs usage to stderr when no arguments or flags provided", func(t *testing.T) {
		pzip := exec.Command(binPath)
		out := testutils.GetOutput(t, pzip)

		assert.Contains(t, out, "punzip is a tool for extracting files concurrently.\n")
		assert.Contains(t, out, "Usage")
	})
	t.Run("extracts an archive", func(t *testing.T) {
		if testing.Short() {
			t.Skip()
		}

		absArchivePath, err := filepath.Abs(archivePath)
		assert.NoError(t, err)

		driver := cli.NewDriver(binPath, absArchivePath, "")

		specifications.Extract(t, driver)
	})
}
