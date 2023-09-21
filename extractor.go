package pzip

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zip"
)

type Extractor struct {
	outputDir string
}

func NewExtractor(outputDir string) *Extractor {
	return &Extractor{outputDir: outputDir}
}

func (e *Extractor) Extract(archivePath string) {
	archiveReader, _ := zip.OpenReader(archivePath)
	defer archiveReader.Close()

	file := archiveReader.File[0]

	if strings.HasSuffix(filepath.ToSlash(file.Name), "/") {
		os.Mkdir(filepath.Join(e.outputDir, file.Name), file.Mode())
	}

	anotherFile := archiveReader.File[1]

	os.Create(filepath.Join(e.outputDir, anotherFile.Name))
}
