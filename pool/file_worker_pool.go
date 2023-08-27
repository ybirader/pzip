package pool

import (
	"errors"
	"sync"

	filebuffer "github.com/pzip/file_buffer"
)

const minNumberOfWorkers = 1

type FileWorkerPool struct {
	tasks           chan filebuffer.File
	executor        func(f filebuffer.File)
	wg              *sync.WaitGroup
	numberOfWorkers int
}

func NewFileWorkerPool(numberOfWorkers int, executor func(f filebuffer.File)) (*FileWorkerPool, error) {
	if numberOfWorkers < minNumberOfWorkers {
		return nil, errors.New("number of workers must be greater than 0")
	}

	return &FileWorkerPool{
		tasks:           make(chan filebuffer.File, 1),
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

func (f *FileWorkerPool) Enqueue(file filebuffer.File) {
	f.tasks <- file
}

func (f *FileWorkerPool) reset() {
	f.tasks = make(chan filebuffer.File)
}
