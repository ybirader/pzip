package pool

import "context"

type WorkerPool[T any] interface {
	Start(ctx context.Context)
	Close() error
	Enqueue(v T)
}
