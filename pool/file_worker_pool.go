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

// A FileWorkerPool is a worker pool in which files are enqueued and for each file, the executor function is called.
// The number of files that can be enqueued for processing at any time is defined by the capacity. The number of
// workers processing files is set by configuring cocnurrency.
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

// Start creates n goroutine workers, where n can be configured by setting
// the concurrency option of the FileWorkerPool. The workers listen and execute tasks
// as they are enqueued. The workers are shut down when an error occurs or the associated
// ctx is canceled.
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

// Enqueue enqueues a file for processing
func (f *FileWorkerPool) Enqueue(file *File) {
	f.tasks <- file
}

// PendingFiles returns the number of tasks that are waiting to be processed
func (f FileWorkerPool) PendingFiles() int {
	return len(f.tasks)
}

// Close gracefully shuts down the FileWorkerPool, ensuring all enqueued tasks have been processed.
// Files cannot be enqueued after Close has been called; attempting this will cause a panic.
// Close returns the first error that was encountered during file processing.
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
