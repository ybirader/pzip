package pzip

import "errors"

const minConcurrency = 1

var (
	ErrMinConcurrency = errors.New("ERROR: concurrency must be 1 or greater")
)

type option func(*archiver) error

// Concurrency sets the number of goroutines used during archiving
// An error is returned if n is less than 1.
func Concurrency(n int) option {
	return func(a *archiver) error {
		if n < minConcurrency {
			return ErrMinConcurrency
		}

		a.concurrency = n
		return nil
	}
}
