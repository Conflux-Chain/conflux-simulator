package main

import (
	"container/heap"
	"math/rand"
	"time"
	"math"
	"sort"
)

type Block struct {
	index      int
	miner      int
	parent     *Block
	references []*Block
	children   []*Block
	seen       map[int]bool
}

type BlockLedger []*Block

type MinerSet struct {
	miners  []*Miner
	weights []float64

	// Initialized when oracle start
	cumtable    []float64
	defaultSeen map[int]bool
}

type Oracle struct {
	queue  *EventQueue
	miners *MinerSet
	blocks BlockLedger

	timestamp  int64
	difficulty float64
	random     *rand.Rand
}

type NetworkManager interface {
	getDelay(int, int) int
}

func NewOracle() *Oracle {
	queue := &EventQueue{queueList: nil}
	heap.Init(queue.queueList)

	miners := &MinerSet{miners: make([]*Miner, 0), weights: make([]float64, 0)}

	difficulty := 1e6
	random := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	return &Oracle{
		queue:      queue,
		miners:     miners,
		timestamp:  0,
		difficulty: difficulty,
		random:     random}
}

func (o *Oracle) run() {
	for {
		event := o.queue.Pop()
		o.timestamp = event.timestamp
		event.Execute(o)
	}
}

func (o *Oracle) getMiner(id int) *Miner {
	return o.miners.miners[id]
}

func (o *Oracle) mineNextBlock() *Event {
	nextstamp := o.timestamp
	for {
		r := o.random.Float64()
		k := math.Log(r) / math.Log(1e-6)
		if k > 3e6 {
			nextstamp = nextstamp + int64(3e6)
		} else {
			nextstamp = nextstamp + int64(math.Ceil(k))
			break
		}
	}

	pickedID := sort.SearchFloat64s(o.miners.cumtable, o.random.Float64())

	event := &Event{timestamp: nextstamp, etype: mineBlock, payload: pickedID}
	return event
}

func (o *Oracle) recordNewBlock(minerID int, block *Block) {
	block.index = len(o.blocks)
	for id, _ := range o.miners.miners {
		block.seen[id] = false
	}
	block.seen[minerID] = true
	block.miner = minerID

	o.blocks = append(o.blocks, block)
}
