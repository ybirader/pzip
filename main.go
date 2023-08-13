package main

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Archiver struct {
	Dest            *os.File
	w               *zip.Writer
	filesToProcess  chan File
	filesToWrite    chan File
	numberOfWorkers int
	fileProcessPool *FileProcessPool
}

type File struct {
	Path string
	Info fs.FileInfo
}

func NewArchiver(archive *os.File) *Archiver {
	a := &Archiver{Dest: archive,
		w:               zip.NewWriter(archive),
		numberOfWorkers: runtime.GOMAXPROCS(0),
	}

	executor := func(file File) {
		a.filesToWrite <- file
	}

	fileProcessPool, _ := NewFileProcessPool(a.numberOfWorkers, executor)
	a.fileProcessPool = fileProcessPool

	return a
}

func (a *Archiver) ArchiveDir(root string) error {
	err := a.newWalkDir(root)

	if err != nil {
		return err
	}

	return nil
}

const minNumberOfWorkers = 1

type FileProcessPool struct {
	tasks           chan File
	executor        func(f File)
	wg              *sync.WaitGroup
	numberOfWorkers int
}

func NewFileProcessPool(numberOfWorkers int, executor func(f File)) (*FileProcessPool, error) {
	if numberOfWorkers < minNumberOfWorkers {
		return nil, errors.New("number of workers must be greater than 0")
	}

	return &FileProcessPool{
		tasks:           make(chan File),
		executor:        executor,
		wg:              new(sync.WaitGroup),
		numberOfWorkers: numberOfWorkers,
	}, nil
}

func (f *FileProcessPool) Start() {
	f.wg.Add(f.numberOfWorkers)
	for i := 0; i < f.numberOfWorkers; i++ {
		go f.listen()
	}
}

func (f *FileProcessPool) Close() {
	close(f.tasks)
	f.wg.Wait()
}

func (f *FileProcessPool) listen() {
	defer f.wg.Done()

	for file := range f.tasks {
		f.executor(file)
	}
}

func (f FileProcessPool) PendingFiles() int {
	return len(f.tasks)
}

func (f *FileProcessPool) Enqueue(file File) {
	f.tasks <- file
}

func (a *Archiver) newWalkDir(root string) error {
	a.filesToWrite = make(chan File)
	a.fileProcessPool.Start()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go a.writeFiles(wg)

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
	close(a.filesToWrite)
	wg.Wait()

	return nil
}

func (a *Archiver) walkDir(root string) error {
	a.initializeChannels()

	wg := new(sync.WaitGroup)
	wg.Add(a.numberOfWorkers)
	awg := new(sync.WaitGroup)

	for i := 0; i < a.numberOfWorkers; i++ {
		go a.processFiles(wg)
	}

	awg.Add(1)
	go a.writeFiles(awg)

	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		f := File{Path: path, Info: info}

		a.filesToProcess <- f

		return nil
	})

	if err != nil {
		return err
	}

	close(a.filesToProcess)
	wg.Wait()
	close(a.filesToWrite)

	awg.Wait()

	return nil
}

func (a *Archiver) ArchiveFiles(files ...string) error {
	a.initializeChannels()

	wg := new(sync.WaitGroup)
	wg.Add(a.numberOfWorkers)
	awg := new(sync.WaitGroup)

	for i := 0; i < a.numberOfWorkers; i++ {
		go a.processFiles(wg)
	}

	awg.Add(1)
	go a.writeFiles(awg)

	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		f := File{Path: path, Info: info}
		a.filesToProcess <- f
	}

	close(a.filesToProcess)
	wg.Wait()
	close(a.filesToWrite)

	awg.Wait()

	return nil
}

func (a *Archiver) Close() error {
	err := a.w.Close()
	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) initializeChannels() {
	a.filesToProcess = make(chan File)
	a.filesToWrite = make(chan File)
}

func (a *Archiver) processFiles(wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range a.filesToProcess {
		a.filesToWrite <- file
	}
}

func (a *Archiver) writeFiles(wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range a.filesToWrite {
		a.archive(&file)
	}
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

func main() {
}
