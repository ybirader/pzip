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
	"golang.org/x/sync/errgroup"
)

type extractor struct {
	outputDir     string
	archiveReader *zip.ReadCloser
	concurrency   int
}

func NewExtractor(outputDir string) *extractor {
	absOutputDir, _ := filepath.Abs(outputDir)

	return &extractor{outputDir: absOutputDir, concurrency: runtime.GOMAXPROCS(0)}
}

func (e *extractor) Extract(ctx context.Context, archivePath string) (err error) {
	e.archiveReader, err = zip.OpenReader(archivePath)
	if err != nil {
		return derrors.Errorf("ERROR: could not read archive at %s: %v", archivePath, err)
	}

	errgroup, ctx := errgroup.WithContext(ctx)
	errgroup.SetLimit(e.concurrency)

	for _, file := range e.archiveReader.File {
		errgroup.Go(func(f *zip.File) func() error {
			return func() error {
				err = e.extractFile(f)
				if err != nil {
					return derrors.Wrapf(err, "ERROR: could not extract file %s", f.Name)
				} else if err := ctx.Err(); err != nil {
					return err
				}
				return nil
			}
		}(file))
	}

	err = errgroup.Wait()

	return err
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

	if err = os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil { // TODO: need to set correct file mode as specified by file
		return derrors.Errorf("ERROR: could not directories %s: %+v", outputPath, err)
	}

	if e.isDir(file.Name) {
		return nil
	}

	outputFile, err := os.Create(e.outputPath(file.Name))
	if err != nil {
		return derrors.Errorf("ERROR: could not create file %s: %v", outputPath, err)
	}
	defer func() {
		err = errors.Join(err, outputFile.Close())
	}()

	fileContent, _ := file.Open()
	defer func() {
		err = errors.Join(err, fileContent.Close())
	}()

	_, err = io.Copy(outputFile, fileContent)
	if err != nil {
		return derrors.Errorf("ERROR: could not decompress file %s", file.Name)
	}

	return nil
}

func (e *extractor) isDir(name string) bool {
	return strings.HasSuffix(filepath.ToSlash(name), "/")
}

func (e *extractor) outputPath(name string) string {
	return filepath.Join(e.outputDir, name)
}
