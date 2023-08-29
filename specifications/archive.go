package specifications

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

type Archiver interface {
	ArchivePath() string
	DirPath() string
	Archive()
}

func ArchiveDir(t *testing.T, driver Archiver) {
	driver.Archive()
	defer os.RemoveAll(driver.ArchivePath())

	assertValidArchive(t, driver.ArchivePath(), driver.DirPath())
}

func assertValidArchive(t testing.TB, archivePath, dirPath string) {
	tmpDirPath, err := os.MkdirTemp("", "unzipped-archive")
	if err != nil {
		t.Fatal("ERROR: could not create temp directory", err)
	}
	defer os.RemoveAll(tmpDirPath)

	unzip := exec.Command("unzip", archivePath, "-d", tmpDirPath)
	unzipOutput, err := unzip.CombinedOutput()

	if err != nil {
		t.Fatalf("ERROR: could not unzip archive %s: %s: %v", archivePath, unzipOutput, err)
	}

	diff := exec.Command("diff", "--recursive", "--brief", dirPath, filepath.Join(tmpDirPath, filepath.Base(dirPath)))
	diffOutput, err := diff.Output()
	if err != nil {
		t.Fatal("ERROR: could not get stdout of diff", err)
	}

	assert.Zero(t, len(diffOutput), fmt.Sprintf("expected no output from diff but got %s", diffOutput))
}
