package main

import (
	"container/heap"
	"math/rand"
	"time"
	"math"
	"sort"
)

type Block struct {
	index    int
	minerID  int
	children []*Block
	seen     map[int]bool
	residual float64

	parent     *Block
	references []*Block
}

type MinerSet struct {
	miners  []*Miner
	weights []float64

	cumtable []float64
}

func (ms *MinerSet) normalize() {
	totalweight := 0.0
	for _, weight := range ms.weights {
		totalweight = totalweight + weight
	}
	ratio := 0.0
	ms.cumtable = make([]float64, len(ms.weights))
	for id, weight := range ms.weights {
		ratio = ratio + weight/totalweight
		ms.cumtable[id] = ratio
	}
}

type Oracle struct {
	queue   *EventQueue
	miners  *MinerSet
	blocks  []*Block
	network *NetworkManager

	timestamp  int64
	difficulty float64
	random     *rand.Rand
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
		o.timestamp = (*event).getTimestamp()
		results := (*event).run(o)
		for _, e := range results {
			if (*e).getTimestamp() >= o.timestamp {
				o.queue.Push(e)
			}
		}
	}
}

func (o *Oracle) lenMiner() int {
	return len(o.miners.miners)
}

func (o *Oracle) getMiner(id int) *Miner {
	return o.miners.miners[id]
}

func (o *Oracle) mineNextBlock() *Event {
	nextstamp := o.timestamp
	threshold := int64(math.Ceil(o.difficulty))
	var residual float64

	for {
		r := o.random.Float64()
		fk := math.Log(r) / (- math.Log(o.difficulty))
		k := int64(math.Ceil(fk))
		residual = float64(k) - fk
		if k > 3*threshold {
			nextstamp = nextstamp + 3*threshold
		} else {
			nextstamp = nextstamp + k
			break
		}
	}

	pickedID := sort.SearchFloat64s(o.miners.cumtable, o.random.Float64())

	block := &Block{
		index:    len(o.blocks),
		minerID:  pickedID,
		residual: residual,
	}
	for id, _ := range o.miners.miners {
		block.seen[id] = false
	}
	o.blocks = append(o.blocks, block)

	if block.parent != nil {
		block.parent.children = append(block.parent.children, block)
	}

	newblockevent := new(Event)
	*newblockevent = &GenBlockEvent{
		BaseEvent: BaseEvent{nextstamp},
		block:     block,
	}
	return newblockevent
}
