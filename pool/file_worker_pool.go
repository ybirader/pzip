package pool

import (
	"errors"
	"sync"
)

const (
	minNumberOfWorkers = 1
	capacity           = 1
)

type FileWorkerPool struct {
	tasks           chan File
	executor        func(f File)
	wg              *sync.WaitGroup
	numberOfWorkers int
}

func NewFileWorkerPool(numberOfWorkers int, executor func(f File)) (*FileWorkerPool, error) {
	if numberOfWorkers < minNumberOfWorkers {
		return nil, errors.New("number of workers must be greater than 0")
	}

	return &FileWorkerPool{
		tasks:           make(chan File, capacity),
		executor:        executor,
		wg:              new(sync.WaitGroup),
		numberOfWorkers: numberOfWorkers,
	}, nil
}

func (f *FileWorkerPool) Start() {
	f.reset()
	f.wg.Add(f.numberOfWorkers)
	for i := 0; i < f.numberOfWorkers; i++ {
		go f.listen()
	}
}

func (f *FileWorkerPool) Close() {
	close(f.tasks)
	f.wg.Wait()
}

func (f *FileWorkerPool) listen() {
	defer f.wg.Done()

	for file := range f.tasks {
		f.executor(file)
	}
}

func (f FileWorkerPool) PendingFiles() int {
	return len(f.tasks)
}

func (f *FileWorkerPool) Enqueue(file File) {
	f.tasks <- file
}

func (f *FileWorkerPool) reset() {
	f.tasks = make(chan File)
}
