package pzip

import "errors"

const minConcurrency = 1

var (
	ErrMinConcurrency = errors.New("ERROR: concurrency must be 1 or greater")
)

type archiverOption func(*archiver) error

// ArchiverConcurrency sets the number of goroutines used during archiving
// An error is returned if n is less than 1.
func ArchiverConcurrency(n int) archiverOption {
	return func(a *archiver) error {
		if n < minConcurrency {
			return ErrMinConcurrency
		}

		a.concurrency = n
		return nil
	}
}
