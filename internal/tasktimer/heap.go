package tasktimer

import (
	"container/heap"
	"time"
)

type scheduleItem struct {
	key        string
	due        time.Time
	generation uint64
}

type scheduleHeap []scheduleItem

func (h scheduleHeap) Len() int {
	return len(h)
}

func (h scheduleHeap) Less(i, j int) bool {
	return h[i].due.Before(h[j].due)
}

func (h scheduleHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *scheduleHeap) Push(x any) {
	*h = append(*h, x.(scheduleItem))
}

func (h *scheduleHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func (h *scheduleHeap) push(item scheduleItem) {
	heap.Push(h, item)
}

func (h *scheduleHeap) pop() scheduleItem {
	return heap.Pop(h).(scheduleItem)
}

func (h *scheduleHeap) init() {
	heap.Init(h)
}
