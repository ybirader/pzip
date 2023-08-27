package pzip

import (
	"os"
)

type CLI struct {
	ArchivePath string
	DirPath     string
}

func (c *CLI) Archive() {
	archive, _ := os.Create(c.ArchivePath)
	defer archive.Close()

	archiver, _ := NewArchiver(archive)
	defer archiver.Close()

	archiver.ArchiveDir(c.DirPath)
}
