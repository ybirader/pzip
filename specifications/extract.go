package specifications

import (
	"os"
	"path/filepath"
	"testing"
)

const testArchiveDirectoryName = "hello"

type Extractor interface {
	DirPath() string
	ArchivePath() string
	Extract()
}

func Extract(t *testing.T, driver Extractor) {
	driver.Extract()
	dirPath := filepath.Join(driver.DirPath(), testArchiveDirectoryName)
	defer os.RemoveAll(dirPath)

	assertValidArchive(t, driver.ArchivePath(), dirPath)
}
