package pzip

import (
	"os"
	"path/filepath"

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

	os.Mkdir(filepath.Join(e.outputDir, file.Name), file.Mode())
}
