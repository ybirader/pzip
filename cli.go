package pzip

import (
	"context"
	"fmt"
	"os"
)

type ArchiverCLI struct {
	ArchivePath string
	Files       []string
	Concurrency int
}

func (a *ArchiverCLI) Archive(ctx context.Context) error {
	archive, err := os.Create(a.ArchivePath)
	if err != nil {
		return fmt.Errorf("create archive at %q: %w", a.ArchivePath, err)
	}
	defer archive.Close()

	archiver, err := NewArchiver(archive, ArchiverConcurrency(a.Concurrency))
	if err != nil {
		return fmt.Errorf("create archiver: %w", err)
	}
	defer archiver.Close()

	err = archiver.Archive(ctx, a.Files)
	if err != nil {
		return fmt.Errorf("archive files: %w", err)
	}

	return nil
}

type ExtractorCLI struct {
	ArchivePath string
	OutputDir   string
	Concurrency int
}

func (e *ExtractorCLI) Extract(ctx context.Context) error {
	extractor, err := NewExtractor(e.OutputDir, ExtractorConcurrency(e.Concurrency))
	if err != nil {
		return fmt.Errorf("new extractor: %w", err)
	}
	defer extractor.Close()

	if err = extractor.Extract(ctx, e.ArchivePath); err != nil {
		return fmt.Errorf("extract %q to %q: %w", e.ArchivePath, e.OutputDir, err)

	}

	return nil
}
