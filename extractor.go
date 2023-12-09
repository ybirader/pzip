package pzip

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zip"
	"github.com/ybirader/pzip/pool"
)

type extractor struct {
	outputDir      string
	archiveReader  *zip.ReadCloser
	fileWorkerPool pool.WorkerPool[zip.File]
	concurrency    int
}

// NewExtractor returns a new pzip extractor. The extractor can be configured by passing in a number of options.
// Available options include ExtractorConcurrency(n int). It returns an error if the extractor can't be created
// Close() should be called on the returned extractor when done
func NewExtractor(outputDir string, options ...extractorOption) (*extractor, error) {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("absolute path %q: %w", outputDir, err)
	}
	e := &extractor{outputDir: absOutputDir, concurrency: runtime.GOMAXPROCS(0)}

	fileExecutor := func(file *zip.File) error {
		if err := e.extractFile(file); err != nil {
			return fmt.Errorf("extract file %q: %w", file.Name, err)
		}

		return nil
	}

	fileWorkerPool, err := pool.NewFileWorkerPool(fileExecutor, &pool.Config{Concurrency: e.concurrency, Capacity: 10})
	if err != nil {
		return nil, fmt.Errorf("new file worker pool: %w", err)
	}

	e.fileWorkerPool = fileWorkerPool

	for _, option := range options {
		if err = option(e); err != nil {
			return nil, err
		}
	}

	return e, nil
}

// Extract extracts the files from the specified archivePath to
// the corresponding outputDir registered with the extractor. Extraction is canceled when the
// associated ctx is canceled. The first error that arises during extraction is returned.
func (e *extractor) Extract(ctx context.Context, archivePath string) (err error) {
	e.archiveReader, err = zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open archive %q: %w", archivePath, err)
	}

	e.fileWorkerPool.Start(ctx)

	for _, file := range e.archiveReader.File {
		e.fileWorkerPool.Enqueue(file)
	}

	if err = e.fileWorkerPool.Close(); err != nil {
		return fmt.Errorf("close file worker pool: %w", err)
	}

	return nil
}

func (e *extractor) Close() error {
	if err := e.archiveReader.Close(); err != nil {
		return fmt.Errorf("close archive reader: %w", err)
	}

	return nil
}

func (e *extractor) extractFile(file *zip.File) (err error) {
	outputPath := e.outputPath(file.Name)

	dir := filepath.Dir(outputPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}

	if e.isDir(file.Name) {
		if err = e.writeDir(outputPath, file); err != nil {
			return fmt.Errorf("write directory %q: %w", file.Name, err)
		}
		return nil
	}

	if err = e.writeFile(outputPath, file); err != nil {
		return fmt.Errorf("write file %q: %w", file.Name, err)
	}

	return nil
}

func (e *extractor) writeDir(outputPath string, file *zip.File) error {
	err := os.Mkdir(outputPath, file.Mode())
	if os.IsExist(err) {
		if err = os.Chmod(outputPath, file.Mode()); err != nil {
			return fmt.Errorf("chmod directory %q: %w", outputPath, err)
		}
	} else if err != nil {
		return fmt.Errorf("create directory %q: %w", outputPath, err)
	}

	return nil
}

func (e *extractor) writeFile(outputPath string, file *zip.File) (err error) {
	outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, file.Mode())
	if err != nil {
		return fmt.Errorf("create file %q: %w", outputPath, err)
	}
	defer func() {
		if cerr := outputFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close output file %q: %w", outputPath, cerr)
		}
	}()

	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("open file %q: %w", file.Name, err)
	}
	defer func() {
		if cerr := srcFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close source file %q: %w", file.Name, cerr)
		}
	}()

	if _, err = io.Copy(outputFile, srcFile); err != nil {
		return fmt.Errorf("decompress file %q: %w", file.Name, err)
	}

	return nil
}

func (e *extractor) isDir(name string) bool {
	return strings.HasSuffix(filepath.ToSlash(name), "/")
}

func (e *extractor) outputPath(name string) string {
	return filepath.Join(e.outputDir, name)
}
