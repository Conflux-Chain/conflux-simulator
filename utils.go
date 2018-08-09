package main

import (
	"container/heap"
)

func UNUSED(v interface{}) {
	_ = v
}

type PriorityQueue []*Event

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return (*pq[i]).getTimestamp() < (*pq[j]).getTimestamp()
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Event)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0: n-1]
	return item
}

type EventQueue struct {
	queueList *PriorityQueue
}

func (eq EventQueue) Push(x *Event) {
	heap.Push(eq.queueList, x)
}

func (eq EventQueue) Pop() *Event {
	return heap.Pop(eq.queueList).(*Event)
}
