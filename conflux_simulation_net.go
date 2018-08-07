package main

import (
	"container/heap"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"sync"
)

// Design:
/*
	1. A priority queue of events
		events type:
		1: SendSigPacket {Sender, SenderTid, Receiver, Seq#, PayloadSigs}
		2: SigReply {Sender, SenderTid}
		3: SendSigTimedOut {Sender, SenderTid, CurrSeq#}
		State:
			Nodes:
				CurrSigs (bool)
				Tid: Seq#=0 (int)
	2. Initial Events
		n*M SigReply {Sender, SenderTid} (null reply to kickoff sender thread)

	3. Processing
		1. SendSigPacket:
			Receiver sig+{}
			Maybe send reply
		2. SigReply:
			Seq++
			(Initiate sending!)
			Send to new random dest {send, recv, seq#, latest sigs}
			(check prob<failure 1%, insert latency 100ms)
		3. Timeout:
			Check seq#; expired -> ignore
			Seq++
			(Initiate sending, same as above)
*/

type Event struct {
	// sorting basis
	timestamp float64
	index     int
	// information
	etype        EventType // 1,2,3
	SenderID     int
	SenderThread int // not used now
	SenderSeq    int // not used now
	ReceiverID   int
	Payload      int
}

type NodeState struct {
	receivedBlocks *Set
	tips           *Set // includes the pivot chain tip
	pivotTip       int
	weight         map[int](int)  // weight for running GHOST
	desc           map[int](*Set) // locally received descendants for blocks
	blocksOnTheFly *Set           // blocks currently being sent to the node, avoid receiving duplicate blocks
	// reservedBlocks *Set
}

// Parameters
var N = 1000

// var M = 32
var generationRate float64 = 0.2
var peerNum = 32
var blockSize float64 = 1               // 1 MByte
var MaxTimestamp float64 = 10000 * 1000 //1000s
var globalLatency float64 = 300         //300ms
var bandwidth float64 = 20              //20 MBps
var nBadNodes = 200                     // number of adversarial node
var blockNum = 360                      // number of generated blocks

// Simulator states
var nodeStates []*NodeState
var parent map[int](int)
var refs map[int](*Set)
var peer map[int]([]int)
var IDCount = 0
var latestTS float64 = -1
var NET_TIME []float64
var badNodes *Set
var withholdedBlock = -1
var prevWithholdedBlock = -1
var withHolding = false
var minedBy map[int](int)

var advPivotCount = 0
var advNonPivotCount = 0
var honestPivotCount = 0
var honestNonPivotCount = 0

// ########################################
// Part 1: Simulator
// ########################################

var rng *rand.Rand // need new RNG each round

type EventType int

const (
	_ EventType = iota
	GENERATE_BLOCK
	RECHECK_PEER
	RECEIVE_BLOCK
)

func addGenerateBlockEvent(currTS float64, node int) {
	ev := &Event{
		timestamp:    currTS,
		etype:        GENERATE_BLOCK,
		SenderID:     node,
		SenderThread: 0,
		SenderSeq:    0,
		ReceiverID:   node,
		Payload:      0,
	}
	insertQueue(ev)
}

func generateBlock(node int) int {
	IDCount++
	blockId := IDCount
	state := nodeStates[node]
	// if true {
	if !badNodes.Has(node) {
		parent[blockId] = state.pivotTip
	} else if withholdedBlock == -1 {
		withholdedBlock = blockId
		parent[blockId] = state.pivotTip
		withHolding = true
	} else {
		if withHolding {
			parent[blockId] = withholdedBlock
		} else {
			parent[blockId] = state.pivotTip
		}
		withholdedBlock = blockId
		withHolding = true
		// fail := false
		// for bro := range state.desc[parent[withholdedBlock]].m {
		// 	if state.weight[bro] > state.weight[withholdedBlock]+1 || (state.weight[bro] == state.weight[withholdedBlock]+1 && bro > withholdedBlock) {
		// 		parent[blockId] = state.pivotTip
		// 		fail = true
		// 		break
		// 	}
		// }
		// if !fail {
		// 	parent[blockId] = withholdedBlock
		// }
		// withholdedBlock = blockId
	}
	blockRefs := NewSet()
	for t := range state.tips.m {
		blockRefs.Add(t)
	}
	refs[blockId] = blockRefs
	minedBy[blockId] = node
	// fmt.Println(node, "Generate block", blockId, parent[blockId], refs[blockId].List(), state.weight)
	// state.receivedBlocks.Add(blockId)
	// state.desc[blockId] = NewSet()
	// state.pivotTip = blockId
	// state.tips.Add(blockId)
	return blockId
}

// Assume every node is always aware of its neighbors' state, and only send a block to one peer who has not received it
func sendBlockToBestPeer(currentTS float64, sender int, blockId int) {
	allHave := true
	nextTime := -1.0
	for _, p := range peer[sender] {
		if !nodeStates[p].receivedBlocks.Has(blockId) && !nodeStates[p].blocksOnTheFly.Has(blockId) {
			allHave = false
			// Only send a block to a neighbor who has received the block's parent
			if nodeStates[p].receivedBlocks.Has(parent[blockId]) {
				// only consider the bandwidth limit on the sender side
				transTime := blockSize * 8 / bandwidth * 1000
				if NET_TIME[sender] <= currentTS {
					NET_TIME[sender] = currentTS + transTime
				} else if NET_TIME[sender]-currentTS > 5000 {
					return
				} else {
					NET_TIME[sender] += transTime
				}
				nodeStates[p].blocksOnTheFly.Add(blockId)
				ev := &Event{
					timestamp:    NET_TIME[sender] + globalLatency,
					etype:        RECEIVE_BLOCK,
					SenderID:     sender,
					SenderThread: 0,
					SenderSeq:    0,
					ReceiverID:   p,
					Payload:      blockId,
				}
				nextTime = NET_TIME[sender] + globalLatency
				insertQueue(ev)
				break
			}
		}
	}

	// some neighbors have not received the block's parent, wait 100ms to check again
	if !allHave {
		if nextTime == -1.0 {
			nextTime = currentTS + 100
		} else {
			nextTime += 1
		}
		ev := &Event{
			timestamp:    nextTime,
			etype:        RECHECK_PEER,
			SenderID:     sender,
			SenderThread: 0,
			SenderSeq:    0,
			ReceiverID:   sender,
			Payload:      blockId,
		}
		insertQueue(ev)
	}
}

// return true if the newly received block can be in pivot chain
func receiveBlock(receiver int, blockId int) bool {
	state := nodeStates[receiver]
	// fmt.Println(receiver, "receive block", blockId, parent[blockId], state.receivedBlocks.List())
	if state.receivedBlocks.Has(blockId) {
		return false
	}

	// Update tip state
	// fmt.Println(receiver, "Receive", blockId)
	state.receivedBlocks.Add(blockId)
	state.blocksOnTheFly.Remove(blockId)
	state.desc[parent[blockId]].Add(blockId)
	state.desc[blockId] = NewSet()
	for t := range refs[blockId].m {
		state.tips.Remove(t)
	}
	state.tips.Remove(parent[blockId])
	state.tips.Add(blockId)

	// Update local block weight
	p := blockId
	for p, ok := parent[p]; ok; p, ok = parent[p] {
		state.weight[p]++
	}

	releaseBlock := false
	// Find the pivot chain
	// 0 is the genesis block
	c := 0
	for !state.desc[c].IsEmpty() {
		max := 0
		max_desc := 0
		advChoice := false
		for d := range state.desc[c].m {
			// FIXME choose the larger block id for the same weight favors later generated blocks
			if d == withholdedBlock {
				advChoice = true
			}
			if state.weight[d] > max || (state.weight[d] == max && d > max_desc) {
				max = state.weight[d]
				max_desc = d
			}
		}
		if advChoice && (state.weight[max_desc] > state.weight[withholdedBlock]+1 ||
			(state.weight[max_desc] == state.weight[withholdedBlock]+1 && withholdedBlock < max_desc)) {
			releaseBlock = true
		}
		c = max_desc
	}
	state.pivotTip = c
	if releaseBlock {
		withHolding = false
	}
	return releaseBlock
}

func procEvent(e *Event) {
	// fmt.Println(e.etype)
	switch e.etype {
	case GENERATE_BLOCK:
		// prevWithholdedBlock := withholdedBlock
		blockId := generateBlock(e.ReceiverID)
		receiveBlock(e.ReceiverID, blockId)
		// if true {
		if !badNodes.Has(e.ReceiverID) {
			sendBlockToBestPeer(e.timestamp, e.ReceiverID, blockId)
		} else {
			for n := range badNodes.m {
				receiveBlock(n, blockId)
				sendBlockToBestPeer(e.timestamp, n, blockId)
			}
			// sendBlockToBestPeer(e.timestamp, e.ReceiverID, blockId)
			// if prevWithholdedBlock != withholdedBlock && prevWithholdedBlock != -1 {
			// 	for n := range badNodes.m {
			// 		sendBlockToBestPeer(e.timestamp, n, prevWithholdedBlock)
			// 	}
			// }
		}
	case RECEIVE_BLOCK:
		receiveBlock(e.ReceiverID, e.Payload)
		// if withholdedBlock != -1 && release && badNodes.Has(e.ReceiverID) {
		// 	for n := range badNodes.m {
		// 		sendBlockToBestPeer(e.timestamp, n, withholdedBlock)
		// 	}
		// 	withholdedBlock = -1
		// }
		if badNodes.Has(e.ReceiverID) {
			for n := range badNodes.m {
				receiveBlock(n, e.Payload)
				// sendBlockToBestPeer(e.timestamp, n, e.Payload)
			}
			return
		}
		sendBlockToBestPeer(e.timestamp, e.ReceiverID, e.Payload)
	case RECHECK_PEER:
		sendBlockToBestPeer(e.timestamp, e.ReceiverID, e.Payload)
	}
}

// actual simulation
func round(seq int, godSeed int, n int) {

	// -------------------------------
	// initialize new simulation

	rng = rand.New(rand.NewSource(int64(godSeed)))
	//clear events
	clearQueue()
	runtime.GC()

	parent = make(map[int](int))
	refs = make(map[int](*Set))
	peer = make(map[int]([]int))
	minedBy = make(map[int](int))
	NET_TIME = make([]float64, n)
	IDCount = 0
	nodeStates = make([]*NodeState, n)
	badNodes = NewSet()
	withholdedBlock = -1
	prevWithholdedBlock = -1

	for i := 0; i < n; i++ {
		nodeStates[i] = &NodeState{}
		nodeStates[i].weight = make(map[int](int))
		nodeStates[i].desc = make(map[int](*Set))
		nodeStates[i].receivedBlocks = NewSet()
		nodeStates[i].tips = NewSet()
		nodeStates[i].desc[0] = NewSet()
		nodeStates[i].blocksOnTheFly = NewSet()
		peer[i] = make([]int, 0)

		nodeStates[i].pivotTip = 0
		nodeStates[i].receivedBlocks.Add(0)
	}
	for i := 0; i < nBadNodes; i++ {
		for {
			n := rng.Intn(n)
			if !badNodes.Has(n) {
				badNodes.Add(n)
				// nodeStates[n].reservedBlocks = NewSet()
				break
			}
		}
	}
	// randomly set up peer connections, but should has the same order after replaying the simulation
	for i := 0; i < n; i++ {
		set := NewSet()
		for _, p := range peer[i] {
			set.Add(p)
		}
		for j := 0; j < peerNum-len(peer[i]); j++ {
			for {
				end := int(rng.Int31n(int32(N)))
				if !set.Has(end) {
					set.Add(end)
					break
				}
			}
		}
		peer[i] = make([]int, 0)
		// change set to array, so iteration returns the same order
		list := set.List()
		sort.Ints(list)
		rand.Shuffle(len(list), func(i, j int) {
			list[i], list[j] = list[j], list[i]
		})
		for _, p := range list {
			peer[i] = append(peer[i], p)
			if p > i {
				peer[p] = append(peer[p], i)
			}
		}
	}
	//init mining events
	var ts float64 = 0.0
	for i := 0; i < blockNum; i++ {
		ts += rng.ExpFloat64() / generationRate * 1000
		addGenerateBlockEvent(ts, int(rng.Int31n(int32(N))))
	}

	// -------------------------------
	// start simulation

	latestTS = -1
	i := 0
	for {
		if queueLen() <= 0 {
			// fmt.Println("Queue Empty!")
			break
		}
		//pop event
		ev := popQueue()
		latestTS = ev.timestamp
		if ev.timestamp > MaxTimestamp {
			break
		}
		//process event
		procEvent(ev)
		if i%10000 == 0 {
			fmt.Fprintf(os.Stderr, "Progress: step=%d ts=%0.3f\n", i, latestTS)
		}
		i++
	}

	// -------------------------------
	// post simulation computation

	computeAntiset()

	// count adversary pivot
	// advPivotCount := 0
	// advNonPivotCount := 0
	// honestPivotCount := 0
	// honestNonPivotCount := 0
	pivot := nodeStates[0].pivotTip
	pivotSet := NewSet()
	for {
		// fmt.Println(pivot)
		pivotSet.Add(pivot)
		if badNodes.Has(minedBy[pivot]) {
			advPivotCount++
		} else {
			honestPivotCount++
		}
		var ok bool
		pivot, ok = parent[pivot]
		// fmt.Println(pivot)
		if pivot == 0 || !ok {
			break
		}
	}
	for i := 1; i <= blockNum; i++ {
		if pivotSet.Has(i) {
			continue
		}
		if badNodes.Has(minedBy[i]) {
			advNonPivotCount++
		} else {
			honestNonPivotCount++
		}
	}
	fmt.Fprintln(os.Stderr, advPivotCount, advNonPivotCount, honestPivotCount, honestNonPivotCount)
	fmt.Fprintf(os.Stderr, "%d %d %.2f\n", n, nBadNodes, latestTS)
}

func computeAntiset() {

	desc := make([]*Set, blockNum+1) // descendants in the DAG (considering both parent link and ref link)
	pre := make([]*Set, blockNum+1)  // predecessors in the DAG
	past := make([]*Set, blockNum+1)
	future := make([]*Set, blockNum+1)
	for i := 0; i <= blockNum; i++ {
		desc[i] = NewSet()
		past[i] = NewSet()
		future[i] = NewSet()
		if i != 0 {
			pre[i] = refs[i]
			pre[i].Add(parent[i])
		} else {
			pre[i] = NewSet()
		}
	}
	for i := 1; i <= blockNum; i++ {
		for r := range pre[i].m {
			desc[r].Add(i)
		}
	}

	// compute past set
	for i := 1; i <= blockNum; i++ {
		s := NewStack()
		accessed := NewSet()
		s.Push(i)
		for {
			data, ok := s.Pop()
			if !ok {
				break
			}
			b := data.(int)
			for p := range pre[b].m {
				if !accessed.Has(p) {
					s.Push(p)
					past[i].Add(p)
					accessed.Add(p)
				}
			}
		}
	}

	// compute future set
	for i := 0; i <= blockNum; i++ {
		s := NewStack()
		accessed := NewSet()
		s.Push(i)
		for {
			data, ok := s.Pop()
			if !ok {
				break
			}
			b := data.(int)
			for d := range desc[b].m {
				if !accessed.Has(d) {
					s.Push(d)
					future[i].Add(d)
					accessed.Add(d)
				}
			}
		}
	}

	// output the size of antiset
	// for i := 0; i <= blockNum; i++ {
	// 	fmt.Println(i, blockNum+1-past[i].Len()-future[i].Len()-1)
	// }
}

// ########################################
// Part 2: Tools
// ########################################
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

// ########################################
// Part 2 END
// ########################################

func main() {
	firstSeed := 3141592
	REP := 10
	n := N
	nBadNodes = 100
	for nBadNodes < n/2 {
		advPivotCount = 0
		advNonPivotCount = 0
		honestPivotCount = 0
		honestNonPivotCount = 0
		for seed := firstSeed; seed < firstSeed+REP; seed += 1 {
			round(seed-firstSeed, seed+n, n)
		}
		fmt.Println(n, nBadNodes, advPivotCount, advNonPivotCount, honestPivotCount, honestNonPivotCount)
		nBadNodes += 100
	}
}
