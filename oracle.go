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
	network *Network

	timestamp     int64
	timePrecision float64
	rate          float64
	duration      int64
}

func NewOracle(timePrecision float64, rate float64, duration float64) *Oracle {
	emptyqueue := make(PriorityQueue, 0)
	queue := &EventQueue{queueList: &emptyqueue}
	heap.Init(queue.queueList)

	miners := &MinerSet{miners: []*Miner{}, weights: []float64{}}

	gensis := &Block{index: 0, minerID: -1, residual: 0, seen: make(map[int]bool), height: 0, ancestorNum: 0}
	blocks := []*Block{gensis}

	return &Oracle{
		queue:         queue,
		miners:        miners,
		blocks:        blocks,
		network:       new(Network),
		timestamp:     0,
		timePrecision: timePrecision,
		duration:      int64(timePrecision * duration),
		rate:          rate,
	}
}

func (o *Oracle) prepare() {
	o.miners.normalize()

	newBlockEvent := o.mineNextBlock()
	o.queue.Push(newBlockEvent)

	BroadcastGensisEvent := new(Event)
	*BroadcastGensisEvent = &BroadcastEvent{
		BaseEvent: BaseEvent{0},
		block:     o.blocks[0],
	}
	o.queue.Push(BroadcastGensisEvent)
}

func (o *Oracle) run() {
	for {
		event := o.queue.Pop();
		o.timestamp = (*event).GetTimestamp()

		if o.timestamp > o.duration {
			break
		}

		results := (*event).Run(o)
		for _, e := range results {
			if (*e).GetTimestamp() >= o.timestamp {
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

func (o *Oracle) getMiner(id int) *Miner {
	return o.miners.miners[id]
}

func (o *Oracle) addHonestMiner(weight float64) {
	ms := o.miners
	index := len(ms.miners)
	miner := new(Miner)
	*miner = NewHonestMiner(index, o)
	ms.miners = append(ms.miners, miner)
	ms.weights = append(ms.weights, weight)
}

func (o *Oracle) SetSimpleNetwork(attacker bool) {
	isAttacker := NewSet()
	if attacker {
		isAttacker.Add(0)
	}
	*o.network = &SimpleNetwork{
		oracle:      o,
		honestDelay: honestDelay,
		attackerIn:  attackerIn,
		attackerOut: attackerOut,
		isAttacker:  isAttacker,
	}
}

func (o *Oracle) SetPeerNetwork() {
	N := len(o.miners.miners)

	peer := make(map[int]([]int))
	sent := make(map[int]*Set)

	for i := 0; i < N; i++ {
		peer[i] = make([]int, 0)
		sent[i] = NewSet()
	}

	// randomly set up peer connections, but should has the same order after replaying the simulation
	for i := 0; i < N; i++ {
		set := NewSet()
		for _, p := range peer[i] {
			set.Add(p)
		}
		for j := 0; j < peers-len(peer[i]); j++ {
			for {
				end := int(rand.Int31n(int32(N)))
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

	*o.network = &PeerNetwork{
		oracle:        o,
		NET_TIME:      make([]float64, N),
		peer:          peer,
		sent:          sent,
		blockSize:     blockSize,
		globalLatency: globalLatency,
		bandwidth:     bandwidth,
	}
}

func (o *Oracle) mineNextBlock() *Event {
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

	pickedID := sort.SearchFloat64s(o.miners.cumtable, rand.Float64())

	block := &Block{
		index:    len(o.blocks),
		minerID:  pickedID,
		residual: residual,
		seen:     make(map[int]bool),
	}
	for id, _ := range o.miners.miners {
		block.seen[id] = false
	}
	o.blocks = append(o.blocks, block)

	newBlockEvent := new(Event)
	*newBlockEvent = &GenBlockEvent{
		BaseEvent: BaseEvent{nextStamp},
		block:     block,
	}
	return newBlockEvent
}
