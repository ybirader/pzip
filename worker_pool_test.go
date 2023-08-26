package pzip

import (
	"bytes"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	filebuffer "github.com/pzip/file_buffer"
)

func TestFileWorkerPool(t *testing.T) {
	t.Run("can enqueue tasks", func(t *testing.T) {
		fileProcessPool := &FileWorkerPool{tasks: make(chan filebuffer.File, 1)}

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

		fileProcessPool, err := NewFileProcessPool(1, executor)
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
		_, err := NewFileProcessPool(0, executor)
		assert.Error(t, err)
	})

	t.Run("can be closed and restarted", func(t *testing.T) {
		output := bytes.Buffer{}
		executor := func(_ filebuffer.File) {
			output.WriteString("hello ")
		}

		fileProcessPool, err := NewFileProcessPool(1, executor)
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
