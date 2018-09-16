package main

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"
)

type Block struct {
	// Maintained by Oracle
	index    int
	minerID  int
	seen     map[int]bool
	residual float64

	// Maintained by Miner
	height      int
	ancestorNum int //The number of ancestorNum doesn't include it self
	parent      *Block
	references  []*Block

	// Maintained by miner of child block
	children    []*Block
	refChildren []*Block

	// Maintained by Receiver
	receivingTime map[int]int64
}

type MinerSet struct {
	miners  []Miner
	weights []float64

	cumTable []float64
}

func (ms *MinerSet) normalize() {
	totalWeight := 0.0
	for _, weight := range ms.weights {
		totalWeight = totalWeight + weight
	}
	ratio := 0.0
	ms.cumTable = make([]float64, len(ms.weights))
	for id, weight := range ms.weights {
		ratio = ratio + weight/totalWeight
		ms.cumTable[id] = ratio
		ms.weights[id] /= totalWeight
	}
}

type Oracle struct {
	queue   *EventQueue
	miners  *MinerSet
	blocks  []*Block
	network Network

	timestamp     int64
	timePrecision float64
	rate          float64
	duration      int64
}

func NewOracle(timePrecision float64, rate float64, duration float64) *Oracle {
	emptyQueue := make(EventPriorityQueue, 0)
	queue := &EventQueue{queueList: &emptyQueue}
	heap.Init(queue.queueList)

	miners := &MinerSet{miners: []Miner{}, weights: []float64{}}

	genesis := &Block{index: 0, minerID: -1, residual: 0, seen: make(map[int]bool), height: 0, ancestorNum: 0, receivingTime: make(map[int]int64)}
	blocks := []*Block{genesis}

	return &Oracle{
		queue:         queue,
		miners:        miners,
		blocks:        blocks,
		network:       nil,
		timestamp:     0,
		timePrecision: timePrecision,
		duration:      int64(timePrecision * duration),
		rate:          rate,
	}
}

func (o *Oracle) prepare() {
	newBlockEvent := o.mineNextBlock()
	o.queue.Push(newBlockEvent)

	broadcastGenesisEvent := &BroadcastEvent{
		BaseEvent: BaseEvent{timestamp: 0},
		block:     o.blocks[0],
	}
	o.queue.Push(broadcastGenesisEvent)
}

func (o *Oracle) run() {
	for {
		event := o.queue.Pop()
		o.timestamp = event.GetTimestamp()

		if o.timestamp > o.duration {
			break
		}

		results := event.Run(o)
		for _, e := range results {
			if e.GetTimestamp() >= o.timestamp {
				o.queue.Push(e)
			}
		}
	}
}

func (o *Oracle) getRealTime() float64 {
	return float64(o.timestamp) / o.timePrecision
}

func (o *Oracle) lenMiner() int {
	return len(o.miners.miners)
}

func (o *Oracle) getMiner(id int) Miner {
	return o.miners.miners[id]
}

func (o *Oracle) addMiner(miner Miner, weight float64) int {
	ms := o.miners
	index := len(ms.miners)
	miner.Setup(o, index)
	ms.miners = append(ms.miners, miner)
	ms.weights = append(ms.weights, weight)

	return index
}

func (o *Oracle) addHonestMiner(weight float64) int {
	miner := NewHonestMiner()
	return o.addMiner(miner, weight)
}

func (o *Oracle) finalizeMiners() {
	o.miners.normalize()
}

func (o *Oracle) setNetwork(network Network) {
	network.Setup(o)
	o.network = network
}

func (o *Oracle) mineNextBlock() Event {
	nextStamp := o.timestamp

	difficulty := o.timePrecision * o.rate
	threshold := 3 * int64(math.Ceil(difficulty))
	var residual float64

	for {
		r := rand.Float64()
		fk := math.Log(r) / (math.Log(1 - 1/difficulty))
		k := int64(math.Ceil(fk))
		residual = float64(k) - fk
		if k > threshold {
			nextStamp = nextStamp + threshold
		} else {
			nextStamp = nextStamp + k
			break
		}
	}

	pickedID := sort.SearchFloat64s(o.miners.cumTable, rand.Float64())

	block := &Block{
		index:         len(o.blocks),
		minerID:       pickedID,
		residual:      residual,
		seen:          make(map[int]bool),
		receivingTime: make(map[int]int64),
	}
	for id := range o.miners.miners {
		block.seen[id] = false
	}
	o.blocks = append(o.blocks, block)

	newBlockEvent := &GenBlockEvent{
		BaseEvent: BaseEvent{timestamp: nextStamp},
		block:     block,
	}
	return newBlockEvent
}
