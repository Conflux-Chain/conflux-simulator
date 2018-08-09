package main

import (
	"container/heap"
	"os"
	"./go-logging"
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

func loadLogger(level logging.Level) {
	formatter := logging.MustStringFormatter(
		"%{color}%{time:15:04:05.0000} [%{level:.4s}] %{shortfile: 19.19s} %{shortfunc: 15.15s} %{module: 5.5s}▶ %{color:reset}%{message} ",
	)
	backend := logging.AddModuleLevel(logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), formatter))
	backend.SetLevel(level, "")

	// Set the backends to be used.
	logging.SetBackend(backend)
}
