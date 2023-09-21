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
	outputDir string
}

func NewExtractor(outputDir string) *Extractor {
	return &Extractor{outputDir: outputDir}
}

func (e *Extractor) Extract(archivePath string) (err error) {
	archiveReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return derrors.Errorf("ERROR: could not read archive at %s: %v", archivePath, err)
	}
	defer func() {
		err = errors.Join(err, archiveReader.Close())
	}()

	for _, file := range archiveReader.File {
		e.extractFile(file)
	}

	return err
}

func (e *Extractor) extractFile(file *zip.File) error {
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
		defer outputFile.Close()

		fileContent, _ := file.Open()
		defer fileContent.Close()
		io.Copy(outputFile, fileContent)
	}

	return nil
}

func (e *Extractor) isDir(name string) bool {
	return strings.HasSuffix(filepath.ToSlash(name), "/")
}

func (e *Extractor) relativeToOutputDir(name string) string {
	return filepath.Join(e.outputDir, name)
}
