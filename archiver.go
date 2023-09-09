package pzip

import (
	"archive/zip"
	"bufio"
	"context"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/pzip/pool"
)

const (
	defaultCompression = -1
	zipVersion20       = 20
	sequentialWrites   = 1
)

const bufferSize = 32 * 1024

var bufferPool = sync.Pool{
	New: func() any {
		return bufio.NewReaderSize(nil, bufferSize)
	},
}

type archiver struct {
	dest            *os.File
	concurrency     int
	w               *zip.Writer
	fileProcessPool pool.WorkerPool[pool.File]
	fileWriterPool  pool.WorkerPool[pool.File]
	chroot          string
}

func NewArchiver(archive *os.File, options ...option) (*archiver, error) {
	a := &archiver{
		dest:        archive,
		w:           zip.NewWriter(archive),
		concurrency: runtime.GOMAXPROCS(0),
	}

	fileProcessExecutor := func(file *pool.File) error {
		err := a.compress(file)
		if err != nil {
			return errors.Wrapf(err, "ERROR: could not compress file %s", file.Path)
		}

		a.fileWriterPool.Enqueue(file)

		return nil
	}

	fileProcessPool, err := pool.NewFileWorkerPool(fileProcessExecutor, &pool.Config{Concurrency: a.concurrency, Capacity: 1})
	if err != nil {
		return nil, errors.Wrap(err, "ERROR: could not create file processor pool")
	}
	a.fileProcessPool = fileProcessPool

	fileWriterExecutor := func(file *pool.File) error {
		err := a.archive(file)
		if err != nil {
			return errors.Wrapf(err, "ERROR: could not write file %s to archive", file.Path)
		}

		return nil
	}

	fileWriterPool, err := pool.NewFileWorkerPool(fileWriterExecutor, &pool.Config{Concurrency: sequentialWrites, Capacity: 1})
	if err != nil {
		return nil, errors.Wrap(err, "ERROR: could not create file writer pool")
	}
	a.fileWriterPool = fileWriterPool

	for _, option := range options {
		err = option(a)
		if err != nil {
			return nil, err
		}
	}

	return a, nil
}

func (a *archiver) Archive(ctx context.Context, filePaths []string) error {
	a.fileProcessPool.Start(ctx)
	a.fileWriterPool.Start(ctx)

	for _, path := range filePaths {
		info, err := os.Lstat(path)
		if err != nil {
			return errors.Errorf("ERROR: could not get stat of %s: %v", path, err)
		}

		if info.IsDir() {
			err = a.ArchiveDir(path)
		} else {
			a.chroot = ""
			file, err := pool.NewFile(path, info, "")
			if err != nil {
				return errors.Wrapf(err, "ERROR: could not create new file %s", path)
			}

			a.ArchiveFile(file)
		}

		if err != nil {
			return errors.Wrapf(err, "ERROR: could not archive %s", path)
		}
	}

	if err := a.fileProcessPool.Close(); err != nil {
		return errors.Wrap(err, "ERROR: could not close file process pool")
	}
	if err := a.fileWriterPool.Close(); err != nil {
		return errors.Wrap(err, "ERROR: could not close file writer pool")
	}

	return nil
}

func (a *archiver) ArchiveDir(root string) error {
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

func (a *archiver) ArchiveFile(file *pool.File) {
	a.fileProcessPool.Enqueue(file)
}

func (a *archiver) Close() error {
	err := a.w.Close()
	if err != nil {
		return errors.New("ERROR: could not close archiver")
	}

	return nil
}

func (a *archiver) changeRoot(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return errors.Errorf("ERROR: could not determine absolute path of %s", root)
	}

	a.chroot = absRoot
	return nil
}

func (a *archiver) walkDir() error {
	err := filepath.Walk(a.chroot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		file, err := pool.NewFile(path, info, a.chroot)
		if err != nil {
			return errors.Wrapf(err, "ERROR: could not create new file %s", path)
		}
		a.ArchiveFile(file)

		return nil
	})

	if err != nil {
		return errors.Errorf("ERROR: could not walk directory %s", a.chroot)
	}

	return nil
}

func (a *archiver) compress(file *pool.File) error {
	var err error

	if file.Info.IsDir() {
		err = a.populateHeader(file)
		if err != nil {
			return errors.Wrapf(err, "ERROR: could not populate file header for %s", file.Path)
		}
		return nil
	}

	hasher := crc32.NewIEEE()

	err = a.copy(io.MultiWriter(file.Compressor, hasher), file)
	if err != nil {
		return errors.Wrapf(err, "ERROR: could not read file %s", file.Path)
	}

	err = file.Compressor.Close()
	if err != nil {
		return errors.New("ERROR: could not close compressor")
	}

	err = a.populateHeader(file)
	if err != nil {
		return errors.Wrapf(err, "ERROR: could not populate file header for %s", file.Path)
	}

	file.Header.CRC32 = hasher.Sum32()
	return nil
}

func (a *archiver) copy(w io.Writer, file *pool.File) error {
	f, err := os.Open(file.Path)
	if err != nil {
		return errors.Errorf("ERROR: could not open file %s", file.Path)
	}
	defer f.Close()

	buf := bufferPool.Get().(*bufio.Reader)
	buf.Reset(f)
	_, err = io.Copy(w, buf)
	bufferPool.Put(buf)
	if err != nil {
		return errors.Errorf("ERROR: could not read file %s: %v", file.Path, err)
	}

	return nil
}

func (a *archiver) populateHeader(file *pool.File) error {
	header := file.Header

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
		header.Name += "/"
		header.Method = zip.Store
		header.Flags &^= 0x8 // won't write data descriptor (crc32, comp, uncomp)
		header.UncompressedSize64 = 0
	} else {
		header.Method = zip.Deflate
		header.Flags |= 0x8 // will write data descriptor (crc32, comp, uncomp)
		header.CompressedSize64 = uint64(file.Written())
	}

	file.Header = header

	return nil
}

func (a *archiver) archive(file *pool.File) error {
	fileWriter, err := a.w.CreateRaw(file.Header)
	if err != nil {
		return errors.Errorf("ERROR: could not write raw header for %s", file.Path)
	}

	_, err = io.Copy(fileWriter, file.CompressedData)
	if err != nil {
		return errors.Errorf("ERROR: could not write content for %s", file.Path)
	}

	if file.Overflowed() {
		file.Overflow.Seek(0, io.SeekStart)
		_, err = io.Copy(fileWriter, file.Overflow)
		if err != nil {
			return errors.Errorf("ERROR: could not write overflow content for %s", file.Path)
		}

		file.Overflow.Close()
		os.Remove(file.Overflow.Name())
	}

	pool.FilePool.Put(file)

	return nil
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
