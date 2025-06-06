package main

import (
	"container/heap"
	"os"
	"./go-logging"
)

func UNUSED(v interface{}) {
	_ = v
}

type EventPriorityQueue []Event

func (pq EventPriorityQueue) Len() int { return len(pq) }

func (pq EventPriorityQueue) Less(i, j int) bool {
	return pq[i].GetTimestamp() < pq[j].GetTimestamp()
}

func (pq EventPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].SetIndex(i)
	pq[j].SetIndex(j)
}

func (pq *EventPriorityQueue) Push(x interface{}) {
	item := x.(Event)
	*pq = append(*pq, item)
	item.SetIndex(len(*pq) - 1)
}

func (pq *EventPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0: n-1]
	return item
}

type PacketEventPriorityQueue []*PacketEvent

func (pq PacketEventPriorityQueue) Len() int { return len(pq) }

func (pq PacketEventPriorityQueue) Less(i, j int) bool {
	return pq[i].accSize < pq[j].accSize
}

func (pq PacketEventPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PacketEventPriorityQueue) Push(x interface{}) {
	item := x.(*PacketEvent)
	*pq = append(*pq, item)
}

func (pq *PacketEventPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0: n-1]
	return item
}

func (pq *PacketEventPriorityQueue) Peek() *PacketEvent {
	old := *pq
	return old[0]
}

type EventQueue struct {
	queueList *EventPriorityQueue
}

func (eq EventQueue) Push(x Event) {
	heap.Push(eq.queueList, x)
	x.SetQueue(&eq)
}

func (eq EventQueue) Pop() Event {
	x := heap.Pop(eq.queueList).(Event)
	x.SetQueue(nil)
	return x
}

func loadLogger(level logging.Level) {
	formatter := logging.MustStringFormatter(
		"%{color}%{time:15:04:05.0000} [%{level:.4s}] %{shortfile: 20.20s} %{shortfunc: 10.10s} %{module: 5.5s}▶ %{color:reset}%{message} ",
	)
	backend := logging.AddModuleLevel(logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), formatter))
	backend.SetLevel(level, "")

	// Set the backends to be used.
	logging.SetBackend(backend)
}

type Stack []interface{}

func NewStack() *Stack {
	result := make(Stack, 0)
	return &result
}

func (s *Stack) Len() int {
	return len(*s)
}

func (s *Stack) Push(item interface{}) {
	*s = append(*s, item)
}

func (s *Stack) Pop() interface{} {
	n := len(*s)
	item := (*s)[n-1]
	*s = (*s)[0:n-1]
	return item
}

func (s *Stack) Peek() interface{} {
	n := len(*s)
	item := (*s)[n-1]
	return item
}

type CountMap map[int]int

func NewCountMap() *CountMap {
	result := make(CountMap)
	return &result
}

func (m *CountMap) Get(id int) int {
	num, ok := (*m)[id]
	if !ok {
		return 0
	}
	return num
}

func (m *CountMap) Incur(id int, num int) {
	oldnum, ok := (*m)[id]
	if !ok {
		(*m)[id] = num
	} else {
		(*m)[id] = oldnum + num
	}
}

func (m *CountMap) Remove(id int) {
	_, ok := (*m)[id]
	if ok {
		delete(*m, id)
	}
}

func (m *CountMap) Merge(n *CountMap) {
	for id, num := range *n {
		m.Incur(id, num)
	}
}

func (m *CountMap) Sum() int {
	answer := 0
	for _, num := range *m {
		answer += num
	}
	return answer
}

type Set struct {
	m map[int]bool
}

func NewSet() *Set {
	return &Set{
		m: map[int]bool{},
	}
}

func (s *Set) Add(item int) {
	s.m[item] = true
}

func (s *Set) Remove(item int) {
	delete(s.m, item)
}

func (s *Set) Has(item int) bool {
	_, ok := s.m[item]
	return ok
}

func (s *Set) Len() int {
	return len(s.List())
}

func (s *Set) Clear() {
	s.m = map[int]bool{}
}

func (s *Set) IsEmpty() bool {
	if s.Len() == 0 {
		return true
	}
	return false
}

func (s *Set) List() []int {
	var list []int
	for item := range s.m {
		list = append(list, item)
	}
	return list
}
