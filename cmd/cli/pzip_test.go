package main_test

import (
	"log"
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
	if testing.Short() {
		t.Skip()
	}

	binPath, cleanup, err := cli.BuildBinary()
	if err != nil {
		log.Fatal("ERROR: could not build binary", err)
	}
	defer cleanup()

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
}
