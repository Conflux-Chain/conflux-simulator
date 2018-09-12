package main

import (
	"sort"
	"math/rand"
)

type BitcoinNetwork struct {
	oracle  *Oracle
	traffic *Traffic

	peers    map[int][]int
	nextTime map[int]int64 // Deprecate Code for FIFO model
	inFlight map[int]*Set
	sent     map[int]*Set
	geo      map[int]int

	attacker   *Set
	verifyTime float64
	relayImpl  int
}

func NewBitcoinNetwork(attacker bool) *BitcoinNetwork {
	isAttacker := NewSet()
	if attacker {
		isAttacker.Add(0)
	}

	network := &BitcoinNetwork{
		verifyTime: 0.01,

		attacker:  isAttacker,
		relayImpl: 0,
	}
	network.traffic = NewTraffic(network)
	return network
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
		for j := 0; j < peers_/2; j++ {
			for {
				end := int(rand.Int31n(int32(N)))
				if !set.Has(end) {
					invN := float64(1.0 / float64(geoN))
					if geo[i] == geo[end] || rand.Float64() < (invN*(1-localRatio_)/localRatio_)/(1-invN) {
						set.Add(end)
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
			peer[p] = append(peer[p], i)
		}
	}

	bn.oracle = o
	bn.peers = peer
	bn.sent = sent
	bn.inFlight = inFlight
	bn.nextTime = nextTime
	bn.geo = geo
}

func (bn *BitcoinNetwork) Broadcast(senderID int, block *Block) []Event {
	bn.sent[senderID].Add(block.index)
	bn.inFlight[senderID].Add(block.index)

	attackerRelay := bn.expressRelay(block)
	if _, ok := block.receivingTime[senderID]; !ok {
		block.receivingTime[senderID] = bn.oracle.timestamp
	}

	return append(bn.sendToAllPeer(senderID, block), attackerRelay...)
}

func (bn *BitcoinNetwork) Relay(senderID int, block *Block) []Event {
	bn.sent[senderID].Add(block.index)
	bn.inFlight[senderID].Add(block.index)

	//For log and statistic
	if _, ok := block.receivingTime[senderID]; !ok {
		block.receivingTime[senderID] = bn.oracle.timestamp

		n := len(bn.oracle.miners.miners)
		if len(block.receivingTime) == n {
			start := block.receivingTime[block.minerID]
			var sum int64
			sum = 0
			for _, item := range block.receivingTime {
				sum += item - start
			}
			avg := float64(sum) / float64(n)
			max := float64(bn.oracle.timestamp - start)

			if block.index%5 == 0 {
				log.Warningf("Block %d, Avg time %0.2f, Max time %0.2f", block.index, avg/bn.oracle.timePrecision, max/bn.oracle.timePrecision)
			} else {
				log.Noticef("Block %d, Avg time %0.2f, Max time %0.2f", block.index, avg/bn.oracle.timePrecision, max/bn.oracle.timePrecision)
			}

		}
	}

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
				size:       32 * bytes,
			},
			blockID: block.index,
		}
		sendINV.childPointer = sendINV
		sendINV.prepare(startTime)
		sendINV.status = sending
		results = append(results, sendINV)
	}

	return results
}

// Interface for attacker network advantage
func (bn *BitcoinNetwork) expressRelay(block *Block) []Event {
	result := make([]Event, 0)
	for _, attacker := range bn.attacker.List() {
		sendEvent := &SendBlockEvent{
			BaseEvent:  BaseEvent{timestamp: bn.oracle.timestamp + 1},
			receiverID: attacker,
			block:      block,
		}
		//log.Criticalf("express block %d", block.index)
		bn.sent[attacker].Add(block.index)
		bn.inFlight[attacker].Add(block.index)
		result = append(result, sendEvent)
	}
	return result
}

func (bn *BitcoinNetwork) expressBroadcast(block *Block) []Event {
	result := make([]Event, 0)
	for receiver := range bn.oracle.miners.miners {
		if receiver == block.minerID {
			continue
		}
		sendEvent := &SendBlockEvent{
			BaseEvent:  BaseEvent{timestamp: bn.oracle.timestamp + 1},
			receiverID: receiver,
			block:      block,
		}
		bn.sent[receiver].Add(block.index)
		bn.inFlight[receiver].Add(block.index)
		result = append(result, sendEvent)
	}
	return result
}

type INVPacketEvent struct {
	PacketEvent
	blockID int
}

func (e *INVPacketEvent) Sent(o *Oracle) []Event {
	network := e.network
	if network.inFlight[e.receiverID].Has(e.blockID) {
		return []Event{}
	}

	result := []Event{}
	switch network.relayImpl {
	case 0:
		getData := &GETPacketEvent{
			PacketEvent: PacketEvent{
				senderID:   e.senderID,
				receiverID: e.receiverID,
				network:    e.network,
				size:       int64(blockSize_ * mb),
			},
			block: o.blocks[e.blockID],
		}
		getData.childPointer = getData

		if getData.senderID == 0 {
			log.Noticef("Time %0.2f, Miner %d request %d", o.getRealTime(), e.receiverID, e.blockID)
		}

		getData.prepare(o.timestamp)
		network.inFlight[e.receiverID].Add(e.blockID)
		result = append(result, getData)
	case 1:
		getData := &GETCompactPacketEvent{
			PacketEvent: PacketEvent{
				senderID:   e.senderID,
				receiverID: e.receiverID,
				network:    e.network,
				size:       int64(blockSize_ * mb / 50),
			},
			block: o.blocks[e.blockID],
		}
		getData.childPointer = getData

		log.Noticef("Time %0.2f, Miner %d request %d", o.getRealTime(), e.receiverID, e.blockID)

		getData.prepare(o.timestamp)
		network.inFlight[e.receiverID].Add(e.blockID)
		result = append(result, getData)
	}
	return result
}

type GETPacketEvent struct {
	PacketEvent
	block *Block
}

func (e *GETPacketEvent) Sent(o *Oracle) []Event {
	receiveEvent := &SendBlockEvent{
		BaseEvent:  BaseEvent{timestamp: e.timestamp},
		block:      e.block,
		receiverID: e.receiverID,
	}
	if e.receiverID == 0 {
		log.Debugf("Relay block %d", e.block.index)
	}
	if e.senderID == 0 {
		log.Noticef("Time %0.2f, Miner %d get block %d", float64(o.timestamp)/timePrecision, e.receiverID, e.block.index)
	}
	return []Event{receiveEvent}
}

type GETCompactPacketEvent struct {
	PacketEvent
	block *Block
}

func (e *GETCompactPacketEvent) Sent(o *Oracle) []Event {
	receiveEvent := &GETPacketEvent{
		PacketEvent: PacketEvent{
			senderID:   e.senderID,
			receiverID: e.receiverID,
			network:    e.network,
			size:       int64(blockSize_ * mb),
		},
		block: e.block,
	}
	receiveEvent.childPointer = receiveEvent
	receiveEvent.prepare(o.timestamp + e.pingDelay())
	if e.receiverID == 0 {
		log.Debugf("Relay block %d", e.block.index)
	}
	if e.senderID == 0 {
		log.Debugf("Receive block %d", e.block.index)
	}
	return []Event{receiveEvent}
}
