package queue

import (
	"iter"
)

type Queue[T any] struct {
	data []T
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

func NewQueueWithCap[T any](cap int) *Queue[T] {
	return &Queue[T]{
		data: make([]T, 0, cap),
	}
}

func (q *Queue[T]) Push(v T) {
	q.data = append(q.data, v)
}

func (q *Queue[T]) mustPop() T {
	v := q.data[0]
	q.data[0] = *new(T) // Drop reference
	q.data = q.data[1:]

	if len(q.data) > 0 && cap(q.data) > 16 && len(q.data) <= cap(q.data)/4 {
		q.shrink()
	}

	return v
}

func (q *Queue[T]) shrink() {
	newData := make([]T, len(q.data), cap(q.data)/2)
	copy(newData, q.data)
	q.data = newData
}

func (q *Queue[T]) Pop() (v T, ok bool) {
	if len(q.data) == 0 {
		ok = false
		return
	}
	return q.mustPop(), true
}

func (q *Queue[T]) Len() int {
	return len(q.data)
}

func (q *Queue[T]) mustPeek() T {
	return q.data[0]
}

func (q *Queue[T]) Peek() (v T, ok bool) {
	if len(q.data) == 0 {
		ok = false
		return
	}
	return q.mustPeek(), true
}

func (q *Queue[T]) Clear() {
	q.data = nil
}

func (q *Queue[T]) Iterator() iter.Seq2[int, T] {
	data := q.data
	return func(yield func(int, T) bool) {
		for k, v := range data {
			if !yield(k, v) {
				break
			}
		}
	}
}

type PriorityQueue[T any] struct {
	data []T
	cmp  func(a, b T) int
}

func NewPriorityQueue[T any](cmp func(a, b T) int) *PriorityQueue[T] {
	return &PriorityQueue[T]{cmp: cmp}
}

func NewPriorityQueueWithCap[T any](cmp func(a, b T) int, cap int) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		data: make([]T, 0, cap),
		cmp:  cmp,
	}
}

func (pq *PriorityQueue[T]) Len() int {
	return len(pq.data)
}

func (pq *PriorityQueue[T]) Push(val T) {
	pq.data = append(pq.data, val)
	pq.up(len(pq.data) - 1)
}

func (pq *PriorityQueue[T]) Pop() (T, bool) {
	if len(pq.data) == 0 {
		var zero T
		return zero, false
	}

	n := len(pq.data) - 1
	pq.data[0], pq.data[n] = pq.data[n], pq.data[0]

	val := pq.data[n]
	pq.data = pq.data[:n]

	if len(pq.data) > 0 {
		pq.down(0)
	}

	if len(pq.data) > 0 && cap(pq.data) > 16 && len(pq.data) <= cap(pq.data)/4 {
		pq.shrink()
	}

	return val, true
}

func (pq *PriorityQueue[T]) Clear() {
	pq.data = nil
}

func (pq *PriorityQueue[T]) Iterator() iter.Seq2[int, T] {
	data := pq.data
	return func(yield func(int, T) bool) {
		for k, v := range data {
			if !yield(k, v) {
				break
			}
		}
	}
}

func (pq *PriorityQueue[T]) shrink() {
	newData := make([]T, len(pq.data), cap(pq.data)/2)
	copy(newData, pq.data)
	pq.data = newData
}

func (pq *PriorityQueue[T]) up(i int) {
	for {
		parent := (i - 1) / 2
		if i == 0 || pq.cmp(pq.data[parent], pq.data[i]) <= 0 {
			break
		}
		pq.data[parent], pq.data[i] = pq.data[i], pq.data[parent]
		i = parent
	}
}

func (pq *PriorityQueue[T]) down(i int) {
	n := len(pq.data)
	for {
		leftChild := 2*i + 1
		if leftChild >= n || leftChild < 0 {
			break
		}

		smallestChild := leftChild
		rightChild := leftChild + 1
		if rightChild < n && pq.cmp(pq.data[rightChild], pq.data[leftChild]) < 0 {
			smallestChild = rightChild
		}

		if pq.cmp(pq.data[i], pq.data[smallestChild]) <= 0 {
			break
		}

		pq.data[i], pq.data[smallestChild] = pq.data[smallestChild], pq.data[i]
		i = smallestChild
	}
}
