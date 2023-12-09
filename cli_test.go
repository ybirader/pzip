package pzip_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/ybirader/pzip"
	"github.com/ybirader/pzip/internal/testutils"
)

const (
	benchmarkRoot            = "testdata/benchmark"
	benchmarkDir             = "minibench"            // modify this to match the file/directory you want to benchmark
	benchmarkArchive         = "miniextractbench.zip" // modify this to match archive you want to benchmark
	testArchiveDirectoryName = "hello"
)

func TestArchiverCLI(t *testing.T) {
	t.Run("archives a directory and some files", func(t *testing.T) {
		files := []string{"testdata/hello", "testdata/hello.txt"}
		archivePath := "testdata/archive.zip"
		defer os.RemoveAll(archivePath)

		cli := pzip.ArchiverCLI{archivePath, files, runtime.GOMAXPROCS(0)}
		err := cli.Archive(context.Background())
		assert.NoError(t, err)

		archiveReader := testutils.GetArchiveReader(t, archivePath)
		defer archiveReader.Close()

		assert.Equal(t, 5, len(archiveReader.File))
	})
}

func TestExtractorCLI(t *testing.T) {
	t.Run("extracts an archive", func(t *testing.T) {
		archivePath := "testdata/test.zip"
		outputDirPath := "testdata/test"

		err := os.Mkdir(outputDirPath, 0755)
		assert.NoError(t, err)
		extractedDirPath := filepath.Join(outputDirPath, testArchiveDirectoryName)
		defer os.RemoveAll(outputDirPath)

		cli := pzip.ExtractorCLI{archivePath, outputDirPath, runtime.GOMAXPROCS(0)}
		err = cli.Extract(context.Background())
		assert.NoError(t, err)

		assert.Equal(t, 3, len(testutils.GetAllFiles(t, extractedDirPath)))
	})
}

// BenchmarkArchiverCLI benchmarks the archiving of a file/directory, referenced by benchmarkDir in the benchmarkRoot directory
func BenchmarkArchiverCLI(b *testing.B) {
	outputDirPath := filepath.Join(benchmarkRoot, benchmarkDir)
	archivePath := filepath.Join(benchmarkRoot, benchmarkDir+".zip")

	cli := pzip.ArchiverCLI{archivePath, []string{outputDirPath}, runtime.GOMAXPROCS(0)}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cli.Archive(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkExtractorCLI benchmarks extracting an archive, referenced by benchmarkArchive
func BenchmarkExtractorCLI(b *testing.B) {
	archivePath := filepath.Join(benchmarkRoot, benchmarkArchive)

	cli := pzip.ExtractorCLI{archivePath, benchmarkRoot, runtime.GOMAXPROCS(0)}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cli.Extract(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}
