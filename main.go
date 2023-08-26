package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/klauspost/compress/flate"
	"github.com/pkg/errors"
)

type Archiver struct {
	Dest            *os.File
	w               *zip.Writer
	numberOfWorkers int
	fileProcessPool *FileWorkerPool
	fileWriterPool  *FileWorkerPool
	chroot          string
}

type File struct {
	Path           string
	Info           fs.FileInfo
	CompressedData bytes.Buffer
	Header         *zip.FileHeader
}

func NewArchiver(archive *os.File) (*Archiver, error) {
	a := &Archiver{Dest: archive,
		w:               zip.NewWriter(archive),
		numberOfWorkers: runtime.GOMAXPROCS(0),
	}

	fileProcessExecutor := func(file File) {
		if !file.Info.IsDir() {
			a.compress(&file)
		}

		a.createHeader(&file)

		a.fileWriterPool.Enqueue(file)
	}

	fileProcessPool, err := NewFileProcessPool(a.numberOfWorkers, fileProcessExecutor)
	if err != nil {
		return nil, errors.Wrap(err, "ERROR: could not create file processor pool")
	}
	a.fileProcessPool = fileProcessPool

	fileWriterExecutor := func(file File) {
		a.archive(&file)
	}

	fileWriterPool, err := NewFileProcessPool(1, fileWriterExecutor)
	if err != nil {
		return nil, errors.Wrap(err, "ERROR: could not create file writer pool")
	}
	a.fileWriterPool = fileWriterPool

	return a, nil
}

func (a *Archiver) changeRoot(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return errors.Errorf("ERROR: could not determine absolute path of %s", root)
	}

	a.chroot = absRoot
	return nil
}

func (a *Archiver) ArchiveDir(root string) error {
	err := a.changeRoot(root)
	if err != nil {
		return errors.Wrapf(err, "ERROR: could not set chroot of archive to %s", root)
	}

	err = a.walkDir()
	if err != nil {
		return errors.Wrap(err, "ERROR: could not walk directory")
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

func (a *Archiver) walkDir() error {
	a.fileProcessPool.Start()
	a.fileWriterPool.Start()

	err := filepath.Walk(a.chroot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == a.chroot {
			return nil
		}

		relativeToRoot, err := filepath.Rel(a.chroot, path)
		if err != nil {
			return errors.Errorf("ERROR: could not determine relative path of %s", path)
		}

		f := File{Path: relativeToRoot, Info: info}
		a.fileProcessPool.Enqueue(f)
		return nil
	})

	if err != nil {
		return errors.Errorf("ERROR: could not walk directory %s", a.chroot)
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
			return errors.Errorf("ERROR: could not get stat of %s", path)
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
		return errors.New("ERROR: could not close archiver")
	}

	return nil
}

func (a *Archiver) archive(f *File) error {
	fileWriter, err := a.w.CreateRaw(f.Header)
	if err != nil {
		return errors.Errorf("ERROR: could not write raw header for %s", f.Path)
	}

	_, err = io.Copy(fileWriter, &f.CompressedData)

	if err != nil {
		return errors.Errorf("ERROR: could not write content for %s", f.Path)
	}

	return nil
}

const DefaultCompression = -1

func (a *Archiver) compress(file *File) error {
	buf := bytes.Buffer{}
	err := a.compressToBuffer(&buf, file)
	if err != nil {
		return errors.Wrapf(err, "ERROR: could not compress to buffer %s", file.Path)
	}
	file.CompressedData = buf
	return nil
}

func (a *Archiver) compressToBuffer(buf *bytes.Buffer, file *File) error {
	f, err := os.Open(file.Path)
	if err != nil {
		return errors.Errorf("ERROR: could not open file %s", file.Path)
	}
	compressor, err := flate.NewWriter(buf, DefaultCompression)
	if err != nil {
		return err
	}

	defer func() {
		cErr := compressor.Close()
		if cErr != nil {
			err = cErr
		}

	}()

	_, err = io.Copy(compressor, f)

	if err != nil {
		return errors.Errorf("ERROR: could not compress file %s", file.Path)
	}

	return nil
}

const zipVersion20 = 20
const extendedTimestampTag = 0x5455

func (a *Archiver) createHeader(file *File) error {
	header, err := zip.FileInfoHeader(file.Info)
	if err != nil {
		return errors.Errorf("ERROR: could not create file header for %s", file.Path)
	}

	if a.dirArchive() {
		header.Name = file.Path
	}

	utf8ValidName, utf8RequireName := detectUTF8(header.Name)
	utf8ValidComment, utf8RequireComment := detectUTF8(header.Comment)
	switch {
	case header.NonUTF8:
		header.Flags &^= 0x800
	case (utf8RequireName || utf8RequireComment) && (utf8ValidName && utf8ValidComment):
		header.Flags |= 0x800
	}

	header.CreatorVersion = header.CreatorVersion&0xff00 | zipVersion20
	header.ReaderVersion = zipVersion20

	// we store local times in header.Modified- other zip readers expect this
	// we set extended timestamp (UTC) info as an Extra for compatibility
	// we only set mod time, not time of last access or time of original creation
	// https://libzip.org/specifications/extrafld.txt

	if !header.Modified.IsZero() {
		header.Extra = append(header.Extra, NewExtendedTimestampExtraField(header.Modified).Encode()...)
	}

	if file.Info.IsDir() {
		header.Method = zip.Store
		header.Flags &^= 0x8 // won't write data descriptor (crc32, comp, uncomp)
		header.UncompressedSize64 = 0
	} else {
		header.Method = zip.Deflate
		header.Flags |= 0x8 // will write data descriptor (crc32, comp, uncomp)
		header.CRC32 = crc32.ChecksumIEEE(file.CompressedData.Bytes())
		header.CompressedSize64 = uint64(file.CompressedData.Len())
	}

	file.Header = header

	return nil
}

type ExtendedTimestampExtraField struct {
	modified time.Time
}

func NewExtendedTimestampExtraField(modified time.Time) *ExtendedTimestampExtraField {
	return &ExtendedTimestampExtraField{
		modified,
	}
}

func (e *ExtendedTimestampExtraField) Encode() []byte {
	extraBuf := make([]byte, 0, 9) // 2*SizeOf(uint16) + SizeOf(uint) + SizeOf(uint32)
	extraBuf = binary.LittleEndian.AppendUint16(extraBuf, extendedTimestampTag)
	extraBuf = binary.LittleEndian.AppendUint16(extraBuf, 5) // block size
	extraBuf = append(extraBuf, uint8(1))                    // flags
	extraBuf = binary.LittleEndian.AppendUint32(extraBuf, uint32(e.modified.Unix()))
	return extraBuf
}

// https://cs.opensource.google/go/go/+/refs/tags/go1.21.0:src/archive/zip/writer.go
func detectUTF8(s string) (valid, require bool) {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		if r < 0x20 || r > 0x7d || r == 0x5c {
			if !utf8.ValidRune(r) || (r == utf8.RuneError && size == 1) {
				return false, false
			}
			require = true
		}
	}
	return true, require
}

func (a *Archiver) dirArchive() bool {
	return a.chroot != ""
}

func main() {
}
