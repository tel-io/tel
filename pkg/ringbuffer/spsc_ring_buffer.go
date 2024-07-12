package ringbuffer

import (
	"sync/atomic"
)

// SpscRingBuffer define Single Producer/Single Consumer ring buffer.
type spscRingBuffer[T any] struct {
	ringbuffer[T]
}

var _ RingBuffer[any] = (*spscRingBuffer[any])(nil)

// NewSpscRingBuffer return the spsc ring buffer with specified capacity.
func New[T any](capacity int) RingBuffer[T] {
	return &spscRingBuffer[T]{
		ringbuffer[T]{
			head:     0,
			tail:     0,
			capacity: capacity,
			elements: make([]T, capacity),
		},
	}
}

// Enqueue element to the ring buffer
// if the ring buffer is full, then return ErrIsFull.
func (q *spscRingBuffer[T]) Enqueue(elem T) error {
	h := atomic.LoadUint64(&q.head)
	t := q.tail
	if t >= h+uint64(q.capacity) {
		return ErrIsFull
	}

	q.elements[t%uint64(q.capacity)] = elem
	atomic.AddUint64(&q.tail, 1)

	return nil
}

// Dequeue an element from the ring buffer
// if the ring buffer is empty, then return ErrIsEmpty.
func (q *spscRingBuffer[T]) Dequeue() (T, error) {
	h := q.head
	t := atomic.LoadUint64(&q.tail)
	if t == h {
		var empty T

		return empty, ErrIsEmpty
	}

	elem := q.elements[h%uint64(q.capacity)]
	atomic.AddUint64(&q.head, 1)

	return elem, nil
}

func (q *spscRingBuffer[T]) Peak() (T, error) {
	h := q.head
	t := atomic.LoadUint64(&q.tail)
	if t == h {
		var empty T

		return empty, ErrIsEmpty
	}

	elem := q.elements[h%uint64(q.capacity)]

	return elem, nil
}
