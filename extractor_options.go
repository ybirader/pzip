package pzip

type extractorOption func(*extractor) error

// ExtractorConcurrency sets the number of goroutines used during extraction
// An error is returned if n is less than 1.
func ExtractorConcurrency(n int) extractorOption {
	return func(e *extractor) error {
		if n < minConcurrency {
			return ErrMinConcurrency
		}

		e.concurrency = n
		return nil
	}
}
