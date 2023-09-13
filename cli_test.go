package pzip_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/pzip"
	"github.com/pzip/internal/testutils"
)

const (
	benchmarkRoot = "testdata/benchmark"
	benchmarkDir  = "minibench" // modify this to match the file/directory you want to benchmark
)

func TestCLI(t *testing.T) {
	t.Run("archives a directory and some files", func(t *testing.T) {
		files := []string{"testdata/hello", "testdata/hello.txt"}
		archivePath := "testdata/archive.zip"
		defer os.RemoveAll(archivePath)

		cli := pzip.CLI{archivePath, files, runtime.GOMAXPROCS(0)}
		err := cli.Archive(context.Background())
		assert.NoError(t, err)

		archiveReader := testutils.GetArchiveReader(t, archivePath)
		defer archiveReader.Close()

		assert.Equal(t, 5, len(archiveReader.File))
	})
}

// BenchmarkCLI benchmarks the archiving of a file/directory, referenced by benchmarkDir in the benchmarkRoot directory
func BenchmarkCLI(b *testing.B) {
	dirPath := filepath.Join(benchmarkRoot, benchmarkDir)
	archivePath := filepath.Join(benchmarkRoot, benchmarkDir+".zip")

	cli := pzip.CLI{archivePath, []string{dirPath}, runtime.GOMAXPROCS(0)}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cli.Archive(context.Background())
	}
}
