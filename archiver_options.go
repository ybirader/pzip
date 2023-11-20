package pzip

import (
	"fmt"
)

const minConcurrency = 1

type archiverOption func(*archiver) error

// ArchiverConcurrency sets the number of goroutines used during archiving
// An error is returned if n is less than 1.
func ArchiverConcurrency(n int) archiverOption {
	return func(a *archiver) error {
		if n < minConcurrency {
			return fmt.Errorf("concurrency %d not greater than zero", n)
		}

		a.concurrency = n
		return nil
	}
}
