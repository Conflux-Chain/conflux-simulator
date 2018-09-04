package main

import (
	"sort"
	"math/rand"
)

type BitcoinNetwork struct {
	oracle *Oracle

	peers    map[int][]int
	nextTime map[int]int64
	inFlight map[int]*Set
	sent     map[int]*Set
	geo      map[int]int

	attacker   *Set
	bandwidth  float64
	verifyTime float64
}

func NewBitcoinNetwork(attacker bool) *BitcoinNetwork {
	isAttacker := NewSet()
	if attacker {
		isAttacker.Add(0)
	}

	return &BitcoinNetwork{
		bandwidth:  bandwidth,
		verifyTime: 0.01,

		attacker: isAttacker,
	}
}

func (bn *BitcoinNetwork) Setup(o *Oracle) {
	bn.oracle = o

	N := len(o.miners.miners)

	peer := make(map[int][]int)
	sent := make(map[int]*Set)
	nextTime := make(map[int]int64)
	inFlight := make(map[int]*Set)
	geo := make(map[int]int)

	for i := 0; i < N; i++ {
		peer[i] = make([]int, 0)
		sent[i] = NewSet()
		inFlight[i] = NewSet()
		nextTime[i] = 0
		geo[i] = rand.Intn(geoN)
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
					invN := float64(1.0/float64(geoN))
					if geo[i] == geo[end] || rand.Float64() < (invN*(1-localRatio)/localRatio)/(1-invN) {
						set.Add(end)
						if i== 0 {
							log.Notice(geo[i] == geo[end])
						}
						break
					}
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

	bn.oracle = o
	bn.peers = peer
	bn.sent = sent
	bn.inFlight = inFlight
	bn.nextTime = nextTime
	bn.geo = geo

	log.Warning("Init Done")
}

func (bn *BitcoinNetwork) Broadcast(senderID int, block *Block) []Event {
	bn.sent[senderID].Add(block.index)
	bn.inFlight[senderID].Add(block.index)

	return bn.sendToAllPeer(senderID, block)
}

func (bn *BitcoinNetwork) Relay(senderID int, block *Block) []Event {
	bn.sent[senderID].Add(block.index)
	bn.inFlight[senderID].Add(block.index)

	return bn.sendToAllPeer(senderID, block)
}

func (bn *BitcoinNetwork) sendToAllPeer(senderID int, block *Block) []Event {
	results := make([]Event, 0)
	startTime := int64(bn.oracle.timePrecision*bn.verifyTime) + bn.oracle.timestamp

	for _, receiverID := range bn.peers[senderID] {
		sendINV := &INVPacketEvent{
			PacketEvent: PacketEvent{
				senderID:   senderID,
				receiverID: receiverID,
				network:    bn,
				size:       32 * byte,
			},
			blockID: block.index,
		}
		sendINV.prepare(startTime)
		results = append(results, sendINV)
	}

	return results
}

type PacketEvent struct {
	BaseEvent
	network    *BitcoinNetwork
	senderID   int
	receiverID int
	size       int
}

const (
	byte = 1
	kb   = 1 << 10
	mb   = 1 << 20
)

func (e *PacketEvent) getDelay() int64 {
	network := e.network
	oracle := network.oracle
	realTime := float64(e.size) / (network.bandwidth * 128 * kb)
	// TODO: Add random here
	return int64(realTime * oracle.timePrecision)
}

func (e *PacketEvent) prepare(startTime int64) {
	network := e.network
	oracle := network.oracle
	nextTime := network.nextTime[e.senderID]
	if nextTime < startTime {
		nextTime = startTime
	}

	sentTime := nextTime + e.getDelay()
	loc1, loc2 := network.geo[e.senderID], network.geo[e.receiverID]
	networkDelay := (geodelay[loc1][loc2] + 4*rand.Float64()) * (0.9 + 0.2*rand.Float64())
	delayOffset := int64(networkDelay / 1000 * oracle.timePrecision)

	network.nextTime[e.senderID] = sentTime
	if e.senderID == 0 {
		log.Debugf("Time %0.2f, miner 0 update finish Time to %0.2f", network.oracle.getRealTime(), float64(sentTime)/network.oracle.timePrecision)
	}
	e.timestamp = sentTime + delayOffset
}

type INVPacketEvent struct {
	PacketEvent
	blockID int
}

func (e *INVPacketEvent) Run(o *Oracle) []Event {
	network := e.network
	if network.inFlight[e.receiverID].Has(e.blockID) {
		return []Event{}
	}

	getData := &GETPacketEvent{
		PacketEvent: PacketEvent{
			senderID:   e.receiverID,
			receiverID: e.senderID,
			network:    e.network,
			size:       blockSize * mb,
		},
		block: o.blocks[e.blockID],
	}

	log.Debugf("Time %0.2f, Miner %d request %d", o.getRealTime(), e.receiverID, e.blockID)

	getData.prepare(o.timestamp)
	network.inFlight[e.receiverID].Add(e.blockID)
	return []Event{getData}
}

type GETPacketEvent struct {
	PacketEvent
	block *Block
}

func (e *GETPacketEvent) Run(o *Oracle) []Event {
	receiveEvent := &SendBlockEvent{
		BaseEvent:  BaseEvent{e.timestamp},
		block:      e.block,
		receiverID: e.senderID,
	}
	return []Event{receiveEvent}
}
