package pzip

import (
	"context"
	"os"

	"github.com/pkg/errors"
)

type ArchiverCLI struct {
	ArchivePath string
	Files       []string
	Concurrency int
}

func (a *ArchiverCLI) Archive(ctx context.Context) error {
	archive, err := os.Create(a.ArchivePath)
	if err != nil {
		return errors.Errorf("ERROR: could not create archive at %s", a.ArchivePath)
	}
	defer archive.Close()

	archiver, err := NewArchiver(archive, ArchiverConcurrency(a.Concurrency))
	if err != nil {
		return errors.Wrap(err, "ERROR: could not create archiver")
	}
	defer archiver.Close()

	err = archiver.Archive(ctx, a.Files)
	if err != nil {
		return errors.Wrapf(err, "ERROR: could not archive files")
	}

	return nil
}

type ExtractorCLI struct {
	ArchivePath string
	DirPath     string
}

func (e *ExtractorCLI) Extract() error {
	extractor, err := NewExtractor(e.DirPath)
	if err != nil {
		return errors.Wrap(err, "ERROR: could not create extractor")
	}
	defer extractor.Close()

	if err = extractor.Extract(context.Background(), e.ArchivePath); err != nil {
		return errors.Wrapf(err, "ERROR: could not extract %s to %s", e.ArchivePath, e.DirPath)

	}

	return nil
}
