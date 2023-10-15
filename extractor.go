package pzip

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zip"
	derrors "github.com/pkg/errors"
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
		return nil, errors.New("ERROR: could not get absoute path of output directory")
	}
	e := &extractor{outputDir: absOutputDir, concurrency: runtime.GOMAXPROCS(0)}

	fileExecutor := func(file *zip.File) error {
		if err := e.extractFile(file); err != nil {
			return derrors.Wrapf(err, "ERROR: could not extract file %s", file.Name)
		}

		return nil
	}

	fileWorkerPool, err := pool.NewFileWorkerPool(fileExecutor, &pool.Config{Concurrency: e.concurrency, Capacity: 10})
	if err != nil {
		return nil, derrors.Wrap(err, "ERROR: could not create new file worker pool")
	}

	e.fileWorkerPool = fileWorkerPool

	for _, option := range options {
		err = option(e)
		if err != nil {
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
		return derrors.Errorf("ERROR: could not read archive at %s: %v", archivePath, err)
	}

	e.fileWorkerPool.Start(ctx)

	for _, file := range e.archiveReader.File {
		e.fileWorkerPool.Enqueue(file)
	}

	if err = e.fileWorkerPool.Close(); err != nil {
		return derrors.Wrap(err, "ERROR: could not close file worker pool")
	}

	return
}

func (e *extractor) Close() error {
	err := e.archiveReader.Close()
	if err != nil {
		return derrors.New("ERROR: could not close archive reader")
	}

	return nil
}

func (e *extractor) extractFile(file *zip.File) (err error) {
	outputPath := e.outputPath(file.Name)

	if err = os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return derrors.Errorf("ERROR: could not directories %s: %+v", outputPath, err)
	}

	if e.isDir(file.Name) {
		if err = e.writeDir(outputPath, file); err != nil {
			return derrors.Wrapf(err, "ERROR: could not write directory %s", file.Name)
		}
		return
	}

	if err = e.writeFile(outputPath, file); err != nil {
		return derrors.Wrapf(err, "ERROR: could not write file %s", file.Name)
	}

	return
}

func (e *extractor) writeDir(outputPath string, file *zip.File) error {
	err := os.Mkdir(outputPath, file.Mode())
	if os.IsExist(err) {
		os.Chmod(outputPath, file.Mode())
		err = nil
	}
	if err != nil {
		return derrors.Errorf("ERROR: could not create directory %s: %+v", file.Name, err)
	}

	return nil
}

func (e *extractor) writeFile(outputPath string, file *zip.File) (err error) {
	outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, file.Mode())
	if err != nil {
		return derrors.Errorf("ERROR: could not create file %s: %v", outputPath, err)
	}
	defer func() {
		err = errors.Join(err, outputFile.Close())
	}()

	srcFile, err := file.Open()
	if err != nil {
		return derrors.Errorf("ERROR: could not open file %s", file.Name)
	}
	defer func() {
		err = errors.Join(err, srcFile.Close())
	}()

	_, err = io.Copy(outputFile, srcFile)
	if err != nil {
		return derrors.Errorf("ERROR: could not decompress file %s", file.Name)
	}

	return
}

func (e *extractor) isDir(name string) bool {
	return strings.HasSuffix(filepath.ToSlash(name), "/")
}

func (e *extractor) outputPath(name string) string {
	return filepath.Join(e.outputDir, name)
}
