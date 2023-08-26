package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/klauspost/compress/flate"
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
		a.compress(&file)
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

func (a *Archiver) changeRoot(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	a.chroot = absRoot
	return nil
}

func (a *Archiver) ArchiveDir(root string) error {
	err := a.changeRoot(root)
	if err != nil {
		return err
	}

	err = a.walkDir()
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
			return err
		}

		f := File{Path: relativeToRoot, Info: info}
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

func (a *Archiver) compress(file *File) error {
	buf := bytes.Buffer{}
	err := a.compressToBuffer(&buf, file)
	if err != nil {
		return err
	}
	file.CompressedData = buf
	return nil
}

func (a *Archiver) compressToBuffer(buf *bytes.Buffer, file *File) error {
	f, err := os.Open(file.Path)
	if err != nil {
		return err
	}
	compressor, err := flate.NewWriter(buf, DefaultCompression)
	if err != nil {
		return err
	}
	defer compressor.Close()
	_, err = io.Copy(compressor, f)
	if err != nil {
		return err
	}

	return nil
}

const zipVersion20 = 20
const extendedTimestampTag = 0x5455

func (a *Archiver) constructHeader(file *File) error {
	header, err := zip.FileInfoHeader(file.Info)
	if err != nil {
		return err
	}

	if a.dirArchive() {
		header.Name = file.Path
	}

	header.Method = zip.Deflate

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

type FileHeader struct {
	// Name is the name of the file.
	//
	// It must be a relative path, not start with a drive letter (such as "C:"),
	// and must use forward slashes instead of back slashes. A trailing slash
	// indicates that this file is a directory and should have no data.
	Name string

	// Comment is any arbitrary user-defined string shorter than 64KiB.
	Comment string

	// NonUTF8 indicates that Name and Comment are not encoded in UTF-8.
	//
	// By specification, the only other encoding permitted should be CP-437,
	// but historically many ZIP readers interpret Name and Comment as whatever
	// the system's local character encoding happens to be.
	//
	// This flag should only be set if the user intends to encode a non-portable
	// ZIP file for a specific localized region. Otherwise, the Writer
	// automatically sets the ZIP format's UTF-8 flag for valid UTF-8 strings.
	NonUTF8 bool

	CreatorVersion uint16
	ReaderVersion  uint16
	Flags          uint16

	// Method is the compression method. If zero, Store is used.
	Method uint16

	// Modified is the modified time of the file.
	//
	// When reading, an extended timestamp is preferred over the legacy MS-DOS
	// date field, and the offset between the times is used as the timezone.
	// If only the MS-DOS date is present, the timezone is assumed to be UTC.
	//
	// When writing, an extended timestamp (which is timezone-agnostic) is
	// always emitted. The legacy MS-DOS date field is encoded according to the
	// location of the Modified time.
	Modified time.Time

	// ModifiedTime is an MS-DOS-encoded time.
	//
	// Deprecated: Use Modified instead.
	ModifiedTime uint16

	// ModifiedDate is an MS-DOS-encoded date.
	//
	// Deprecated: Use Modified instead.
	ModifiedDate uint16

	// CRC32 is the CRC32 checksum of the file content.
	CRC32 uint32

	// CompressedSize is the compressed size of the file in bytes.
	// If either the uncompressed or compressed size of the file
	// does not fit in 32 bits, CompressedSize is set to ^uint32(0).
	//
	// Deprecated: Use CompressedSize64 instead.
	CompressedSize uint32

	// UncompressedSize is the compressed size of the file in bytes.
	// If either the uncompressed or compressed size of the file
	// does not fit in 32 bits, CompressedSize is set to ^uint32(0).
	//
	// Deprecated: Use UncompressedSize64 instead.
	UncompressedSize uint32

	// CompressedSize64 is the compressed size of the file in bytes.
	CompressedSize64 uint64

	// UncompressedSize64 is the uncompressed size of the file in bytes.
	UncompressedSize64 uint64

	Extra         []byte
	ExternalAttrs uint32 // Meaning depends on CreatorVersion
}

func main() {
}
