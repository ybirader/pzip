package main_test

import (
	"log"
	"path/filepath"
	"testing"

	"github.com/pzip/adapters/cli"
	"github.com/pzip/specifications"
)

const (
	testdataRoot         = "../../testdata"
	archivePath          = testdataRoot + "/archive.zip"
	dirPath              = testdataRoot + "/hello"
	benchmarkRoot        = testdataRoot + "/benchmark"
	benchmarkDirPath     = benchmarkRoot + "/bench"
	benchmarkArchivePath = benchmarkRoot + "/bench.zip"
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

func BenchmarkPzip(b *testing.B) {
	binPath, cleanup, err := cli.BuildBinary()
	if err != nil {
		log.Fatal("ERROR: could not build binary", err)
	}
	defer cleanup()

	absArchivePath, err := filepath.Abs(benchmarkArchivePath)
	if err != nil {
		b.Fatalf("ERROR: could not get path to archive %s", benchmarkArchivePath)
	}

	absDirPath, err := filepath.Abs(benchmarkDirPath)
	if err != nil {
		b.Fatalf("ERROR: could not get path to directory %s", benchmarkDirPath)
	}

	driver := cli.NewDriver(binPath, absArchivePath, absDirPath)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		driver.Archive()
	}
}
