package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/klauspost/compress/flate"
)

type Archiver struct {
	Dest            *os.File
	w               *zip.Writer
	numberOfWorkers int
	fileProcessPool *FileWorkerPool
	fileWriterPool  *FileWorkerPool
}

type File struct {
	Path string
	Info fs.FileInfo
}

func NewArchiver(archive *os.File) (*Archiver, error) {
	a := &Archiver{Dest: archive,
		w:               zip.NewWriter(archive),
		numberOfWorkers: runtime.GOMAXPROCS(0),
	}

	fileProcessExecutor := func(file File) {
		a.fileWriterPool.Enqueue(file)
	}

	fileProcessPool, err := NewFileProcessPool(a.numberOfWorkers, fileProcessExecutor)
	if err != nil {
		return nil, err
	}
	a.fileProcessPool = fileProcessPool

	fileWriterExecutor := func(file File) {
		a.archive(&file)
	}

	fileWriterPool, err := NewFileProcessPool(1, fileWriterExecutor)
	if err != nil {
		return nil, err
	}
	a.fileWriterPool = fileWriterPool

	return a, nil
}

func (a *Archiver) ArchiveDir(root string) error {
	err := a.walkDir(root)

	if err != nil {
		return err
	}

	return nil
}

const minNumberOfWorkers = 1

type FileWorkerPool struct {
	tasks           chan File
	executor        func(f File)
	wg              *sync.WaitGroup
	numberOfWorkers int
}

func NewFileProcessPool(numberOfWorkers int, executor func(f File)) (*FileWorkerPool, error) {
	if numberOfWorkers < minNumberOfWorkers {
		return nil, errors.New("number of workers must be greater than 0")
	}

	return &FileWorkerPool{
		tasks:           make(chan File),
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

func (a *Archiver) walkDir(root string) error {
	a.fileProcessPool.Start()
	a.fileWriterPool.Start()

	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		f := File{Path: path, Info: info}
		a.fileProcessPool.Enqueue(f)
		return nil
	})

	if err != nil {
		return err
	}

	a.fileProcessPool.Close()
	a.fileWriterPool.Close()

	return nil
}

func (a *Archiver) ArchiveFiles(files ...string) error {
	a.fileProcessPool.Start()
	a.fileWriterPool.Start()

	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		f := File{Path: path, Info: info}
		a.fileProcessPool.Enqueue(f)
	}

	a.fileProcessPool.Close()
	a.fileWriterPool.Close()

	return nil
}

func (a *Archiver) Close() error {
	err := a.w.Close()
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) archive(f *File) error {
	err := a.writeFile(f)

	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) writeFile(f *File) error {
	writer, err := a.createFile(f.Info)
	if err != nil {
		return err
	}

	if f.Info.IsDir() {
		return nil
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}

	err = a.writeContents(writer, file)
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) createFile(info fs.FileInfo) (io.Writer, error) {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return nil, err
	}

	writer, err := a.w.CreateHeader(header)
	if err != nil {
		return nil, err
	}

	return writer, nil
}

func (a *Archiver) writeContents(w io.Writer, r io.Reader) error {
	_, err := io.Copy(w, r)
	if err != nil {
		return err
	}

	return nil
}

const DefaultCompression = -1

func compressToBuffer(buf *bytes.Buffer, file File) {
	f, _ := os.Open(file.Path)
	compressor, _ := flate.NewWriter(buf, DefaultCompression)
	defer compressor.Close()
	io.Copy(compressor, f)
}

func main() {
}
