package net

import (
	"container/heap"
	"sync"
)

type PriorityQueue []*Event

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].timestamp < pq[j].timestamp
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Event)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

var QUEUE PriorityQueue

func queueLen() int {
	return len(QUEUE)
}

func clearQueue() {
	for queueLen() > 0 {
		_ = popQueue()
	}
	QUEUE = nil
	heap.Init(&QUEUE)
}

func insertQueue(e *Event) {
	heap.Push(&QUEUE, e)
}

func popQueue() (e *Event) {
	return heap.Pop(&QUEUE).(*Event)
}

type Set struct {
	m map[int]bool
	sync.RWMutex
}

func NewSet() *Set {
	return &Set{
		m: map[int]bool{},
	}
}

func (s *Set) Add(item int) {
	s.Lock()
	defer s.Unlock()
	s.m[item] = true
}

func (s *Set) Remove(item int) {
	s.Lock()
	s.Unlock()
	delete(s.m, item)
}

func (s *Set) Has(item int) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.m[item]
	return ok
}

func (s *Set) Len() int {
	return len(s.List())
}

func (s *Set) Clear() {
	s.Lock()
	defer s.Unlock()
	s.m = map[int]bool{}
}

func (s *Set) IsEmpty() bool {
	if s.Len() == 0 {
		return true
	}
	return false
}

func (s *Set) List() []int {
	s.RLock()
	defer s.RUnlock()
	list := []int{}
	for item := range s.m {
		list = append(list, item)
	}
	return list
}

type Node struct {
	data interface{}
	next *Node
}

type Stack struct {
	head *Node
}

func NewStack() *Stack {
	s := &Stack{nil}
	return s
}

func (s *Stack) Push(data interface{}) {
	n := &Node{data: data, next: s.head}
	s.head = n
}

func (s *Stack) Pop() (interface{}, bool) {
	n := s.head
	if s.head == nil {
		return nil, false
	}
	s.head = s.head.next
	return n.data, true
}