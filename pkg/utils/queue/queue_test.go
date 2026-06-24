/*
LLM usage: this test file was generated with deepseek-v4-pro with manual modifications.
*/
package queue

import (
	"cmp"
	"slices"
	"testing"
)

// =============================================================================
// Queue tests
// =============================================================================

func TestQueue_PushPop(t *testing.T) {
	q := NewQueue[int]()

	q.Push(1)
	q.Push(2)
	q.Push(3)

	if q.Len() != 3 {
		t.Fatalf("expected len 3, got %d", q.Len())
	}

	v, ok := q.Pop()
	if !ok || v != 1 {
		t.Fatalf("expected (1, true), got (%d, %v)", v, ok)
	}
	v, ok = q.Pop()
	if !ok || v != 2 {
		t.Fatalf("expected (2, true), got (%d, %v)", v, ok)
	}
	v, ok = q.Pop()
	if !ok || v != 3 {
		t.Fatalf("expected (3, true), got (%d, %v)", v, ok)
	}

	if q.Len() != 0 {
		t.Fatalf("expected len 0, got %d", q.Len())
	}
}

func TestQueue_PopEmpty(t *testing.T) {
	q := Queue[string]{}

	_, ok := q.Pop()
	if ok {
		t.Fatal("expected Pop on empty queue to return ok=false")
	}
}

func TestQueue_Peek(t *testing.T) {
	q := NewQueue[int]()
	q.Push(10)
	q.Push(20)

	v, ok := q.Peek()
	if !ok || v != 10 {
		t.Fatalf("expected (10, true), got (%d, %v)", v, ok)
	}
	// Peek again — still the same element
	v, ok = q.Peek()
	if !ok || v != 10 {
		t.Fatalf("expected (10, true) on second peek, got (%d, %v)", v, ok)
	}
}

func TestQueue_PeekEmpty(t *testing.T) {
	q := Queue[float64]{}

	_, ok := q.Peek()
	if ok {
		t.Fatal("expected Peek on empty queue to return ok=false")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue[int]()
	q.Push(1)
	q.Push(2)
	q.Clear()

	if q.Len() != 0 {
		t.Fatalf("expected len 0 after Clear, got %d", q.Len())
	}

	_, ok := q.Pop()
	if ok {
		t.Fatal("expected Pop after Clear to return ok=false")
	}
}

func TestQueue_Iterator(t *testing.T) {
	q := NewQueue[int]()
	q.Push(1)
	q.Push(2)
	q.Push(3)

	var got []int
	for k, v := range q.Iterator() {
		got = append(got, v)
		if k == 1 {
			break // test early termination
		}
	}

	if !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("expected [1, 2] after early break, got %v", got)
	}
}

func TestQueue_IteratorFull(t *testing.T) {
	q := Queue[string]{}
	q.Push("a")
	q.Push("b")
	q.Push("c")

	var got []string
	for _, v := range q.Iterator() {
		got = append(got, v)
	}

	if !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("expected [a, b, c], got %v", got)
	}
}

func TestQueue_IteratorEmpty(t *testing.T) {
	q := NewQueue[int]()

	count := 0
	for range q.Iterator() {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 iterations on empty queue, got %d", count)
	}
}

func TestQueue_PushPopAlternating(t *testing.T) {
	q := NewQueue[int]()

	q.Push(1)
	q.Push(2)
	v, _ := q.Pop() // 1
	if v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}

	q.Push(3)
	v, _ = q.Pop() // 2
	if v != 2 {
		t.Fatalf("expected 2, got %d", v)
	}
	v, _ = q.Pop() // 3
	if v != 3 {
		t.Fatalf("expected 3, got %d", v)
	}

	if q.Len() != 0 {
		t.Fatalf("expected empty queue, got len %d", q.Len())
	}
}

func TestQueue_Shrink(t *testing.T) {
	q := NewQueue[int]()

	// Push enough elements to build up capacity.
	const n = 200
	for i := range n {
		q.Push(i)
	}

	// Pop most of them; once len < cap/4 and cap > 16, shrink should
	// trigger on the next Pop.
	for range n - 5 {
		q.Pop()
	}

	// After the shrink the queue must still hold the remaining elements.
	if q.Len() != 5 {
		t.Fatalf("expected len 5 after pops, got %d", q.Len())
	}
	for i := range 5 {
		v, ok := q.Pop()
		if !ok || v != n-5+i {
			t.Fatalf("expected %d, got (%d, %v)", n-5+i, v, ok)
		}
	}
}

func TestQueue_ZeroValue(t *testing.T) {
	var q Queue[int]
	// All methods should work on a zero-value Queue.

	if q.Len() != 0 {
		t.Fatal("expected zero-value queue to have len 0")
	}

	v, ok := q.Pop()
	if ok || v != 0 {
		t.Fatal("expected Pop on zero-value queue to return zero, false")
	}

	v, ok = q.Peek()
	if ok || v != 0 {
		t.Fatal("expected Peek on zero-value queue to return zero, false")
	}
}

func TestQueue_StringType(t *testing.T) {
	q := Queue[string]{}
	q.Push("hello")
	q.Push("world")

	v, ok := q.Peek()
	if !ok || v != "hello" {
		t.Fatalf("expected (hello, true), got (%s, %v)", v, ok)
	}

	v, ok = q.Pop()
	if !ok || v != "hello" {
		t.Fatalf("expected (hello, true), got (%s, %v)", v, ok)
	}
	v, ok = q.Pop()
	if !ok || v != "world" {
		t.Fatalf("expected (world, true), got (%s, %v)", v, ok)
	}
}

// =============================================================================
// PriorityQueue tests
// =============================================================================

func TestPriorityQueue_PushPop(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	pq.Push(3)
	pq.Push(1)
	pq.Push(2)

	if pq.Len() != 3 {
		t.Fatalf("expected len 3, got %d", pq.Len())
	}

	v, ok := pq.Pop()
	if !ok || v != 1 {
		t.Fatalf("expected (1, true), got (%d, %v)", v, ok)
	}
	v, ok = pq.Pop()
	if !ok || v != 2 {
		t.Fatalf("expected (2, true), got (%d, %v)", v, ok)
	}
	v, ok = pq.Pop()
	if !ok || v != 3 {
		t.Fatalf("expected (3, true), got (%d, %v)", v, ok)
	}

	if pq.Len() != 0 {
		t.Fatalf("expected len 0, got %d", pq.Len())
	}
}

func TestPriorityQueue_PopEmpty(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[string])

	v, ok := pq.Pop()
	if ok || v != "" {
		t.Fatalf("expected (\"\", false), got (%q, %v)", v, ok)
	}
}

func TestPriorityQueue_SortedOrder(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	input := []int{7, 2, 9, 1, 5, 3, 8, 4, 6}
	for _, v := range input {
		pq.Push(v)
	}

	var got []int
	for pq.Len() > 0 {
		v, ok := pq.Pop()
		if !ok {
			t.Fatal("unexpected empty Pop")
		}
		got = append(got, v)
	}

	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if !slices.Equal(got, expected) {
		t.Fatalf("expected sorted %v, got %v", expected, got)
	}
}

func TestPriorityQueue_Duplicates(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	pq.Push(5)
	pq.Push(3)
	pq.Push(3)
	pq.Push(1)
	pq.Push(3)

	var got []int
	for pq.Len() > 0 {
		v, _ := pq.Pop()
		got = append(got, v)
	}

	expected := []int{1, 3, 3, 3, 5}
	if !slices.Equal(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestPriorityQueue_Interleaved(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])

	pq.Push(5)
	pq.Push(2)
	v, _ := pq.Pop()
	if v != 2 {
		t.Fatalf("expected 2, got %d", v)
	}

	pq.Push(1)
	pq.Push(4)
	v, _ = pq.Pop()
	if v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
	v, _ = pq.Pop()
	if v != 4 {
		t.Fatalf("expected 4, got %d", v)
	}
	v, _ = pq.Pop()
	if v != 5 {
		t.Fatalf("expected 5, got %d", v)
	}
}

func TestPriorityQueue_Shrink(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])

	const n = 200
	for i := range n {
		pq.Push(i)
	}

	// Pop most of them; once len < cap/4 and cap > 16, shrink should
	// trigger.
	for range n - 5 {
		pq.Pop()
	}

	if pq.Len() != 5 {
		t.Fatalf("expected len 5, got %d", pq.Len())
	}
	// Remaining elements must still be the largest ones, in order.
	for i := range 5 {
		v, ok := pq.Pop()
		if !ok || v != n-5+i {
			t.Fatalf("expected %d, got (%d, %v)", n-5+i, v, ok)
		}
	}
}

func TestPriorityQueue_ZeroValue(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])

	if pq.Len() != 0 {
		t.Fatal("expected fresh PriorityQueue to have len 0")
	}

	v, ok := pq.Pop()
	if ok || v != 0 {
		t.Fatal("expected Pop on empty PriorityQueue to return zero, false")
	}
}

func TestPriorityQueue_SingleElement(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	pq.Push(42)

	if pq.Len() != 1 {
		t.Fatalf("expected len 1, got %d", pq.Len())
	}

	v, ok := pq.Pop()
	if !ok || v != 42 {
		t.Fatalf("expected (42, true), got (%d, %v)", v, ok)
	}

	_, ok = pq.Pop()
	if ok {
		t.Fatal("expected second Pop to return false")
	}
}

func TestPriorityQueue_StringType(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[string])
	pq.Push("zebra")
	pq.Push("apple")
	pq.Push("mango")

	v, _ := pq.Pop()
	if v != "apple" {
		t.Fatalf("expected apple, got %s", v)
	}
	v, _ = pq.Pop()
	if v != "mango" {
		t.Fatalf("expected mango, got %s", v)
	}
	v, _ = pq.Pop()
	if v != "zebra" {
		t.Fatalf("expected zebra, got %s", v)
	}
}

func TestPriorityQueue_Large(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])

	const n = 10000
	for i := n; i > 0; i-- { // push in reverse order
		pq.Push(i)
	}

	prev := -1
	for pq.Len() > 0 {
		v, ok := pq.Pop()
		if !ok {
			t.Fatal("unexpected empty Pop")
		}
		if v < prev {
			t.Fatalf("heap invariant violated: %d < %d", v, prev)
		}
		prev = v
	}
	if prev != n {
		t.Fatalf("last element should be %d, got %d", n, prev)
	}
}

func TestPriorityQueue_Clear(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	pq.Push(3)
	pq.Push(1)
	pq.Push(2)
	pq.Clear()

	if pq.Len() != 0 {
		t.Fatalf("expected len 0 after Clear, got %d", pq.Len())
	}

	_, ok := pq.Pop()
	if ok {
		t.Fatal("expected Pop after Clear to return ok=false")
	}
}

func TestPriorityQueue_Iterator(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	pq.Push(3)
	pq.Push(1)
	pq.Push(2)

	// Iterator yields elements in heap-array order (not sorted).
	var got []int
	for k, v := range pq.Iterator() {
		got = append(got, v)
		if k == 1 {
			break // test early termination
		}
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 elements after early break, got %d: %v", len(got), got)
	}

	// Heap with [1, 3, 2]: index 0 -> 1, index 1 -> 3 — so first two are 1,3.
	if got[0] != 1 || got[1] != 3 {
		t.Fatalf("expected [1, 3] (heap order), got %v", got)
	}
}

func TestPriorityQueue_IteratorFull(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	pq.Push(3)
	pq.Push(1)
	pq.Push(2)

	var got []int
	for _, v := range pq.Iterator() {
		got = append(got, v)
	}

	// Heap with elements 1,2,3: internal order is [1, 3, 2].
	if !slices.Equal(got, []int{1, 3, 2}) {
		t.Fatalf("expected [1, 3, 2] (heap order), got %v", got)
	}
}

func TestPriorityQueue_IteratorEmpty(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])

	count := 0
	for range pq.Iterator() {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 iterations on empty priority queue, got %d", count)
	}

	// Clear then iterate.
	pq.Push(1)
	pq.Clear()
	for range pq.Iterator() {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 iterations after Clear, got %d", count)
	}
}

func TestPriorityQueue_ClearThenReuse(t *testing.T) {
	pq := NewPriorityQueue(cmp.Compare[int])
	pq.Push(5)
	pq.Push(3)
	pq.Push(4)
	pq.Clear()

	// Reuse after Clear.
	pq.Push(10)
	pq.Push(1)

	v, _ := pq.Pop()
	if v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
	v, _ = pq.Pop()
	if v != 10 {
		t.Fatalf("expected 10, got %d", v)
	}
}
