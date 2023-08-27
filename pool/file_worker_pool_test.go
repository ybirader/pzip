package pool_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	filebuffer "github.com/pzip/file_buffer"
	"github.com/pzip/pool"
)

const (
	testdataRoot             = "../testdata/"
	archivePath              = testdataRoot + "archive.zip"
	helloTxtFileFixture      = testdataRoot + "hello.txt"
	helloMarkdownFileFixture = testdataRoot + "hello.md"
	helloDirectoryFixture    = testdataRoot + "hello/"
)

func TestFileWorkerPool(t *testing.T) {
	t.Run("can enqueue tasks", func(t *testing.T) {
		fileProcessPool, err := pool.NewFileWorkerPool(1, func(f filebuffer.File) {})
		assert.NoError(t, err)

		info := getFileInfo(t, helloTxtFileFixture)
		fileProcessPool.Enqueue(filebuffer.File{Path: helloTxtFileFixture, Info: info})

		assert.Equal(t, 1, fileProcessPool.PendingFiles())
	})

	t.Run("has workers process files to completion", func(t *testing.T) {
		output := bytes.Buffer{}
		executor := func(_ filebuffer.File) {
			time.Sleep(5 * time.Millisecond)
			output.WriteString("hello, world!")
		}

		fileProcessPool, err := pool.NewFileWorkerPool(1, executor)
		assert.NoError(t, err)
		fileProcessPool.Start()

		info := getFileInfo(t, helloTxtFileFixture)
		fileProcessPool.Enqueue(filebuffer.File{Path: helloTxtFileFixture, Info: info})

		fileProcessPool.Close()

		assert.Equal(t, 0, fileProcessPool.PendingFiles())
		assert.Equal(t, "hello, world!", output.String())
	})

	t.Run("returns an error if number of workers is less than one", func(t *testing.T) {
		executor := func(_ filebuffer.File) {
		}
		_, err := pool.NewFileWorkerPool(0, executor)
		assert.Error(t, err)
	})

	t.Run("can be closed and restarted", func(t *testing.T) {
		output := bytes.Buffer{}
		executor := func(_ filebuffer.File) {
			output.WriteString("hello ")
		}

		fileProcessPool, err := pool.NewFileWorkerPool(1, executor)
		assert.NoError(t, err)
		fileProcessPool.Start()

		info := getFileInfo(t, helloTxtFileFixture)
		fileProcessPool.Enqueue(filebuffer.File{Path: helloTxtFileFixture, Info: info})

		fileProcessPool.Close()

		fileProcessPool.Start()
		info = getFileInfo(t, helloTxtFileFixture)
		fileProcessPool.Enqueue(filebuffer.File{Path: helloTxtFileFixture, Info: info})

		fileProcessPool.Close()

		assert.Equal(t, "hello hello ", output.String())
	})
}

func getFileInfo(t testing.TB, name string) fs.FileInfo {
	t.Helper()

	info, err := os.Stat(name)
	assert.NoError(t, err, fmt.Sprintf("could not get file into fot %s", name))

	return info
}
