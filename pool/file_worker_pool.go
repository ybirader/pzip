package pool

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const (
	minConcurrency = 1
)

type Config struct {
	Concurrency int
	Capacity    int
}

type FileWorkerPool struct {
	tasks       chan *File
	executor    func(f *File) error
	g           *errgroup.Group
	ctxCancel   func(error)
	concurrency int
	capacity    int
}

func NewFileWorkerPool(executor func(f *File) error, config *Config) (*FileWorkerPool, error) {
	if config.Concurrency < minConcurrency {
		return nil, errors.New("number of workers must be greater than 0")
	}

	return &FileWorkerPool{
		tasks:       make(chan *File, config.Capacity),
		executor:    executor,
		g:           new(errgroup.Group),
		concurrency: config.Concurrency,
		capacity:    config.Capacity,
	}, nil
}

func (f *FileWorkerPool) Start(ctx context.Context) {
	f.reset()

	ctx, cancel := context.WithCancelCause(ctx)
	f.ctxCancel = cancel

	for i := 0; i < f.concurrency; i++ {
		f.g.Go(func() error {
			if err := f.listen(ctx); err != nil {
				f.ctxCancel(err)
				return err
			}

			return nil
		})
	}
}

func (f *FileWorkerPool) Enqueue(file *File) {
	f.tasks <- file
}

func (f FileWorkerPool) PendingFiles() int {
	return len(f.tasks)
}

func (f *FileWorkerPool) Close() error {
	close(f.tasks)
	err := f.g.Wait()
	f.ctxCancel(err)
	return err
}

func (f *FileWorkerPool) listen(ctx context.Context) error {
	for file := range f.tasks {
		if err := f.executor(file); err != nil {
			return errors.Wrapf(err, "ERROR: could not process file %s", file.Path)
		} else if err := ctx.Err(); err != nil {
			return err
		}
	}

	return nil
}

func (f *FileWorkerPool) reset() {
	f.tasks = make(chan *File, f.capacity)
}
