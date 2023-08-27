package pool

type WorkerPool[T any] interface {
	Start()
	Close()
	Enqueue(v T)
}
