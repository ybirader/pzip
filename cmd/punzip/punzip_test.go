package main_test

import (
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/pzip/adapters/cli"
	"github.com/pzip/specifications"
)

const (
	testdataRoot = "../../testdata"
	archivePath  = testdataRoot + "/test.zip"
)

func TestPunzip(t *testing.T) {
	t.Run("extracts an archive", func(t *testing.T) {
		binPath, cleanup, err := cli.BuildBinary()
		if err != nil {
			t.Fatal("ERROR: could not build binary", err)
		}
		t.Cleanup(cleanup)

		absArchivePath, err := filepath.Abs(archivePath)
		assert.NoError(t, err)

		driver := cli.NewDriver(binPath, absArchivePath, "")

		specifications.Extract(t, driver)
	})
}
