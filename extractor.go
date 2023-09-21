package pzip

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zip"
	derrors "github.com/pkg/errors"
)

type Extractor struct {
	outputDir     string
	archiveReader *zip.ReadCloser
}

func NewExtractor(outputDir string) *Extractor {
	return &Extractor{outputDir: outputDir}
}

func (e *Extractor) Extract(archivePath string) (err error) {
	e.archiveReader, err = zip.OpenReader(archivePath)
	if err != nil {
		return derrors.Errorf("ERROR: could not read archive at %s: %v", archivePath, err)
	}

	for _, file := range e.archiveReader.File {
		err = e.extractFile(file)
		if err != nil {
			return derrors.Wrapf(err, "ERROR: could not extract file %s", file.Name)
		}
	}

	return err
}

func (e *Extractor) Close() error {
	err := e.archiveReader.Close()
	if err != nil {
		return derrors.New("ERROR: could not close archive reader")
	}

	return nil
}

func (e *Extractor) extractFile(file *zip.File) (err error) {
	pathRelativeToRoot := e.relativeToOutputDir(file.Name)

	if e.isDir(file.Name) {
		err := os.Mkdir(pathRelativeToRoot, file.Mode())
		if err != nil {
			return derrors.Errorf("ERROR: could not create directory %s: %v", pathRelativeToRoot, err)
		}
	} else {
		outputFile, err := os.Create(e.relativeToOutputDir(file.Name))
		if err != nil {
			return derrors.Errorf("ERROR: could not create file %s: %v", pathRelativeToRoot, err)
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
	}

	return nil
}

func (e *Extractor) isDir(name string) bool {
	return strings.HasSuffix(filepath.ToSlash(name), "/")
}

func (e *Extractor) relativeToOutputDir(name string) string {
	return filepath.Join(e.outputDir, name)
}
