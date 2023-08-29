package pzip

import (
	"archive/zip"
	"bytes"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"unicode/utf8"

	"github.com/klauspost/compress/flate"
	"github.com/pkg/errors"
	"github.com/pzip/pool"
)

const (
	defaultCompression = -1
	zipVersion20       = 20
)

type Archiver struct {
	Dest            *os.File
	w               *zip.Writer
	numberOfWorkers int
	fileProcessPool pool.WorkerPool[pool.File]
	fileWriterPool  pool.WorkerPool[pool.File]
	chroot          string
}

func NewArchiver(archive *os.File) (*Archiver, error) {
	a := &Archiver{Dest: archive,
		w:               zip.NewWriter(archive),
		numberOfWorkers: runtime.GOMAXPROCS(0),
	}

	fileProcessExecutor := func(file pool.File) {
		hdr, _ := zip.FileInfoHeader(file.Info)
		file.Header = hdr

		if !file.Info.IsDir() {
			a.compress(&file)
		}

		a.createHeader(&file)

		a.fileWriterPool.Enqueue(file)
	}

	fileProcessPool, err := pool.NewFileWorkerPool(a.numberOfWorkers, fileProcessExecutor)
	if err != nil {
		return nil, errors.Wrap(err, "ERROR: could not create file processor pool")
	}
	a.fileProcessPool = fileProcessPool

	fileWriterExecutor := func(file pool.File) {
		a.archive(&file)
	}

	fileWriterPool, err := pool.NewFileWorkerPool(1, fileWriterExecutor)
	if err != nil {
		return nil, errors.Wrap(err, "ERROR: could not create file writer pool")
	}
	a.fileWriterPool = fileWriterPool

	return a, nil
}

func (a *Archiver) Archive(files []string) error {
	a.fileProcessPool.Start()
	a.fileWriterPool.Start()

	for _, file := range files {
		info, err := os.Lstat(file)
		if err != nil {
			return errors.Errorf("ERROR: could not get stat of %s: %v", file, err)
		}

		if info.IsDir() {
			err = a.ArchiveDir(file)
		} else {
			f := pool.File{Path: file, Info: info}
			a.ArchiveFile(f)
		}

		if err != nil {
			return errors.Wrapf(err, "ERROR: could not archive %s", file)
		}
	}

	a.fileProcessPool.Close()
	a.fileWriterPool.Close()

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

func (a *Archiver) ArchiveFile(f pool.File) {
	a.fileProcessPool.Enqueue(f)
}

func (a *Archiver) Close() error {
	err := a.w.Close()
	if err != nil {
		return errors.New("ERROR: could not close archiver")
	}

	return nil
}

func (a *Archiver) changeRoot(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return errors.Errorf("ERROR: could not determine absolute path of %s", root)
	}

	a.chroot = absRoot
	return nil
}

func (a *Archiver) walkDir() error {
	err := filepath.Walk(a.chroot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativeToRoot, err := filepath.Rel(a.chroot, path)
		relativeIncludingRoot := filepath.Join(filepath.Base(a.chroot), relativeToRoot)

		if err != nil {
			return errors.Errorf("ERROR: could not determine relative path of %s", path)
		}

		f := pool.File{Name: relativeIncludingRoot, Path: path, Info: info}
		a.ArchiveFile(f)

		return nil
	})

	if err != nil {
		return errors.Errorf("ERROR: could not walk directory %s", a.chroot)
	}

	return nil
}

func (a *Archiver) compress(file *pool.File) error {
	buf := bytes.Buffer{}
	err := a.compressToBuffer(&buf, file)
	if err != nil {
		return errors.Wrapf(err, "ERROR: could not compress to buffer %s", file.Path)
	}
	file.CompressedData = buf
	return nil
}

func (a *Archiver) compressToBuffer(buf *bytes.Buffer, file *pool.File) error {
	f, err := os.Open(file.Path)
	if err != nil {
		return errors.Errorf("ERROR: could not open file %s", file.Path)
	}

	compressor, err := flate.NewWriter(buf, defaultCompression)
	if err != nil {
		return err
	}

	defer func() {
		cErr := compressor.Close()
		if cErr != nil {
			err = cErr
		}
	}()

	hasher := crc32.NewIEEE()

	writer := io.MultiWriter(compressor, hasher)

	_, err = io.Copy(writer, f)

	file.Header.CRC32 = hasher.Sum32()

	if err != nil {
		return errors.Errorf("ERROR: could not compress file %s", file.Path)
	}

	return nil
}

func (a *Archiver) createHeader(file *pool.File) error {
	header := file.Header

	if a.dirArchive() {
		header.Name = file.Name
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
		header.Name += "/"
		header.Method = zip.Store
		header.Flags &^= 0x8 // won't write data descriptor (crc32, comp, uncomp)
		header.UncompressedSize64 = 0
	} else {
		header.Method = zip.Deflate
		header.Flags |= 0x8 // will write data descriptor (crc32, comp, uncomp)
		header.CompressedSize64 = uint64(file.CompressedData.Len())
	}

	file.Header = header

	return nil
}

func (a *Archiver) dirArchive() bool {
	return a.chroot != ""
}

func (a *Archiver) archive(f *pool.File) error {
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
