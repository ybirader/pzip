package main

import (
	"archive/zip"
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
	numberOfWorkers int
}

type File struct {
	Path string
	Info fs.FileInfo
}

func NewArchiver(archive *os.File) *Archiver {
	return &Archiver{Dest: archive, w: zip.NewWriter(archive), numberOfWorkers: runtime.GOMAXPROCS(0)}
}

func (a *Archiver) ArchiveDir(root string) error {
	err := a.walkDir(root)

	if err != nil {
		return err
	}

	return nil
}

func (a *Archiver) walkDir(root string) error {
	filesToProcess := make(chan File)
	filesToWrite := make(chan File)
	wg := new(sync.WaitGroup)
	wg.Add(a.numberOfWorkers)

	awg := new(sync.WaitGroup)

	for i := 0; i < a.numberOfWorkers; i++ {
		go a.processFiles(filesToProcess, filesToWrite, wg)
	}

	awg.Add(1)
	go a.writeFiles(filesToWrite, awg)

	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root {
			return nil
		}

		f := File{Path: path, Info: info}

		filesToProcess <- f

		return nil
	})

	if err != nil {
		return err
	}

	close(filesToProcess)
	wg.Wait()
	close(filesToWrite)

	awg.Wait()

	return nil
}

func (a *Archiver) ArchiveFiles(files ...string) error {
	filesToProcess := make(chan File)
	filesToWrite := make(chan File)
	wg := new(sync.WaitGroup)
	wg.Add(a.numberOfWorkers)

	awg := new(sync.WaitGroup)

	for i := 0; i < a.numberOfWorkers; i++ {
		go a.processFiles(filesToProcess, filesToWrite, wg)
	}

	awg.Add(1)
	go a.writeFiles(filesToWrite, awg)

	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		f := File{Path: path, Info: info}
		filesToProcess <- f
	}

	close(filesToProcess)
	wg.Wait()
	close(filesToWrite)

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

func (a *Archiver) processFiles(filesToProcess <-chan File, filesToWrite chan<- File, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range filesToProcess {
		filesToWrite <- file
	}
}

func (a *Archiver) writeFiles(filesToWrite <-chan File, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range filesToWrite {
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
