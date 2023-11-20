package pzip

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unicode/utf8"

	"github.com/ybirader/pzip/pool"
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

// NewArchiver returns a new pzip archiver. The archiver can be configured by passing in a number of options.
// Available options include ArchiverConcurrency(n int). It returns an error if the archiver can't be created
// Close() should be called on the returned archiver when done
func NewArchiver(archive *os.File, options ...archiverOption) (*archiver, error) {
	a := &archiver{
		dest:        archive,
		w:           zip.NewWriter(archive),
		concurrency: runtime.GOMAXPROCS(0),
	}

	fileProcessExecutor := func(file *pool.File) error {
		err := a.compress(file)
		if err != nil {
			return fmt.Errorf("compress file %q: %w", file.Path, err)
		}

		a.fileWriterPool.Enqueue(file)

		return nil
	}

	fileProcessPool, err := pool.NewFileWorkerPool(fileProcessExecutor, &pool.Config{Concurrency: a.concurrency, Capacity: 1})
	if err != nil {
		return nil, fmt.Errorf("new file process pool: %w", err)
	}
	a.fileProcessPool = fileProcessPool

	fileWriterExecutor := func(file *pool.File) error {
		err := a.archive(file)
		if err != nil {
			return fmt.Errorf("archive %q: %w", file.Path, err)
		}

		return nil
	}

	fileWriterPool, err := pool.NewFileWorkerPool(fileWriterExecutor, &pool.Config{Concurrency: sequentialWrites, Capacity: 1})
	if err != nil {
		return nil, fmt.Errorf("new file writer pool: %w", err)
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

// Archive compresses and stores (archives) the files at the provides filePaths to
// the corresponding archive registered with the archiver. Archiving is canceled when the
// associated ctx is canceled. The first error that arises during archiving is returned.
func (a *archiver) Archive(ctx context.Context, filePaths []string) error {
	a.fileProcessPool.Start(ctx)
	a.fileWriterPool.Start(ctx)

	for _, path := range filePaths {
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("lstat %q: %w", path, err)
		}

		if info.IsDir() {
			if err = a.archiveDir(path); err != nil {
				return fmt.Errorf("archive dir %q: %w", path, err)
			}
		} else {
			a.chroot = ""
			file, err := pool.NewFile(path, info, "")
			if err != nil {
				return fmt.Errorf("new file %q: %w", path, err)
			}

			a.archiveFile(file)
		}
	}

	if err := a.fileProcessPool.Close(); err != nil {
		return fmt.Errorf("close file process pool: %w", err)
	}

	if err := a.fileWriterPool.Close(); err != nil {
		return fmt.Errorf("close file writer pool: %w", err)
	}

	return nil
}

func (a *archiver) Close() error {
	if err := a.w.Close(); err != nil {
		return fmt.Errorf("close zip writer: %w", err)
	}

	return nil
}

func (a *archiver) archiveDir(root string) error {
	if err := a.changeRoot(root); err != nil {
		return fmt.Errorf("change root to %q: %w", root, err)
	}

	if err := a.walkDir(); err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	return nil
}

func (a *archiver) archiveFile(file *pool.File) {
	a.fileProcessPool.Enqueue(file)
}

func (a *archiver) changeRoot(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("get absolute path of %q: %w", root, err)
	}

	a.chroot = absRoot
	return nil
}

func (a *archiver) walkDir() error {
	if err := filepath.Walk(a.chroot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		file, err := pool.NewFile(path, info, a.chroot)
		if err != nil {
			return fmt.Errorf("new file %q: %w", path, err)
		}
		a.archiveFile(file)

		return nil
	}); err != nil {
		return fmt.Errorf("walk directory %q: %w", a.chroot, err)
	}

	return nil
}

func (a *archiver) compress(file *pool.File) error {
	if file.Info.IsDir() {
		if err := a.populateHeader(file); err != nil {
			return fmt.Errorf("populate header for %q: %w", file.Path, err)
		}
		return nil
	}

	hasher := crc32.NewIEEE()

	if err := a.copy(io.MultiWriter(file.Compressor, hasher), file); err != nil {
		return fmt.Errorf("copy %q: %w", file.Path, err)
	}

	if err := file.Compressor.Close(); err != nil {
		return fmt.Errorf("close compressor for %q: %w", file.Path, err)
	}

	if err := a.populateHeader(file); err != nil {
		return fmt.Errorf("populate header for %q: %w", file.Path, err)
	}

	file.Header.CRC32 = hasher.Sum32()
	return nil
}

func (a *archiver) copy(w io.Writer, file *pool.File) error {
	f, err := os.Open(file.Path)
	if err != nil {
		return fmt.Errorf("open %q: %w", file.Path, err)
	}
	defer f.Close()

	buf := bufferPool.Get().(*bufio.Reader)
	buf.Reset(f)

	_, err = io.Copy(w, buf)
	bufferPool.Put(buf)
	if err != nil {
		return fmt.Errorf("copy %q: %w", file.Path, err)
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
		return fmt.Errorf("create raw for %q: %w", file.Path, err)
	}

	if _, err = io.Copy(fileWriter, file.CompressedData); err != nil {
		return fmt.Errorf("write compressed data for %q: %w", file.Path, err)
	}

	if file.Overflowed() {
		if _, err = file.Overflow.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("seek overflow for %q: %w", file.Path, err)
		}
		if _, err = io.Copy(fileWriter, file.Overflow); err != nil {
			return fmt.Errorf("copy overflow for %q: %w", file.Path, err)
		}

		file.Overflow.Close()
		if err = os.Remove(file.Overflow.Name()); err != nil {
			return fmt.Errorf("remove overflow for %q: %w", file.Overflow.Name(), err)
		}
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
