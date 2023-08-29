package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func BuildBinary() (binPath string, cleanup func(), err error) {
	binName := "pzip-test"

	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	build := exec.Command("go", "build", "-o", binName)

	if err := build.Run(); err != nil {
		return "", nil, err
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}

	binPath = filepath.Join(dir, binName)

	cleanup = func() {
		os.Remove(binPath)
	}

	return
}
