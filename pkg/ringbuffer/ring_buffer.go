package ringbuffer

import (
	"errors"
	"sync/atomic"
	"unsafe"
)

var (
	// ErrIsEmpty indicate ring buffer is empty.
	ErrIsEmpty = errors.New("ring buffer is empty")
	// ErrIsFull indicate ring buffer is full.
	ErrIsFull = errors.New("ring buffer is full")
)

// RingBuffer is an interface.
type RingBuffer[T any] interface {
	Enqueue(T) error
	Dequeue() (T, error)
	Peak() (T, error)
	Length() int
	Capacity() int
}

type cacheLinePad struct {
	_ [128 - unsafe.Sizeof(uint64(0))%128]byte
}

// ringbuffer struct.
//
//nolint:structcheck
type ringbuffer[T any] struct {
	head     uint64
	_        cacheLinePad
	tail     uint64
	_        cacheLinePad
	capacity int
	elements []T
}

// Length return the number of all elements.
func (q *ringbuffer[T]) Length() int {
	return int(atomic.LoadUint64(&q.tail) - atomic.LoadUint64(&q.head))
}

// Capacity return the capacity of ring buffer.
func (q *ringbuffer[T]) Capacity() int {
	return q.capacity
}
