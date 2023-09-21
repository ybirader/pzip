package cli

import (
	"log"
	"os/exec"
)

type Driver struct {
	binPath     string
	archivePath string
	dirPath     string
}

func NewDriver(binPath, archivePath, dirPath string) *Driver {
	return &Driver{binPath, archivePath, dirPath}
}

func (d *Driver) DirPath() string {
	return d.dirPath
}

func (d *Driver) ArchivePath() string {
	return d.archivePath
}

func (d *Driver) Archive() {
	pzip := exec.Command(d.binPath, d.ArchivePath(), d.DirPath())

	if err := pzip.Run(); err != nil {
		log.Fatal("ERROR: could not run pzip binary", err)
	}
}

func (d *Driver) Extract() {
	punzip := exec.Command(d.binPath, d.ArchivePath())

	if err := punzip.Run(); err != nil {
		log.Fatal("ERROR: could not run punzip binary", err)
	}
}
