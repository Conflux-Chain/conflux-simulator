package main

import (
	"sort"
	"math/rand"
)

type PeerNetwork struct {
	oracle *Oracle

	sent     map[int]*Set
	peer     map[int][]int
	NET_TIME []float64

	blockSize     float64
	bandwidth     float64
	globalLatency float64

	attacker    *Set
	attackerIn  float64
	attackerOut float64

	startTime map[int]int64 //For Log Only
	endTime   map[int]int64
}

func NewPeerNetwork(attacker bool) *PeerNetwork {
	isAttacker := NewSet()
	if attacker {
		isAttacker.Add(0)
	}

	return &PeerNetwork{
		blockSize:     blockSize_,
		globalLatency: globalLatency,
		bandwidth:     bandwidth_,

		attacker:    isAttacker,
		attackerIn:  attackerIn,
		attackerOut: attackerOut,

		startTime: make(map[int]int64),
		endTime:   make(map[int]int64),
	}
}

func (pn *PeerNetwork) Setup(o *Oracle) {
	pn.oracle = o

	N := len(o.miners.miners)

	peer := make(map[int][]int)
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
		for j := 0; j < peers_-len(peer[i]); j++ {
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

	pn.oracle = o
	pn.NET_TIME = make([]float64, N)
	pn.peer = peer
	pn.sent = sent
}

func (pn *PeerNetwork) Broadcast(id int, block *Block) []Event {
	pn.startTime[block.index] = pn.oracle.timestamp
	pn.endTime[block.index] = pn.oracle.timestamp

	result := make([]Event, 0)
	if pn.attacker.Has(block.minerID) {
		result1 := pn.expressBroadcast(block)
		result = append(result, result1...)
	}
	result2 := pn.expressRelay(block)
	eventSent := &PeerSendEvent{
		BaseEvent: BaseEvent{timestamp: pn.oracle.timestamp},
		senderID:  id,
		block:     block,
		network:   pn,
		first:     true,
	}
	result = append(result, result2...)
	result = append(result, eventSent)
	return result
}

func (pn *PeerNetwork) Relay(id int, block *Block) []Event {
	sendToPeer := &PeerSendEvent{
		BaseEvent: BaseEvent{timestamp: pn.oracle.timestamp},
		senderID:  id,
		block:     block,
		network:   pn,
		first:     false,
	}
	return []Event{sendToPeer}
}

type PeerSendEvent struct {
	BaseEvent
	senderID int
	block    *Block
	network  *PeerNetwork
	first    bool
}

func (e *PeerSendEvent) Run(o *Oracle) ([]Event) {
	log.Debugf("PeerSend  Event: time %.2f, block %d, sender %d", o.getRealTime(), e.block.index, e.senderID)
	return e.network.sendBlockToBestPeer(e)
}

func (pn *PeerNetwork) toTimestamp(t float64) int64 {
	return int64(t * float64(pn.oracle.timePrecision))
}

func (pn *PeerNetwork) sendBlockToBestPeer(e *PeerSendEvent) []Event {
	var result []Event
	sender := e.senderID
	block := e.block
	currentTS := pn.oracle.getRealTime()

	allHave := true
	nextTime := -1.0

	for _, p := range pn.peer[sender] {
		peerMap := pn.sent[p]
		if !peerMap.Has(block.index) {
			allHave = false
			transTime := pn.blockSize * 8 / pn.bandwidth * 1.0
			if pn.NET_TIME[sender] <= currentTS {
				pn.NET_TIME[sender] = currentTS + transTime
			} else if pn.NET_TIME[sender]-currentTS > 5.0 && ! e.first {
				break
			} else {
				pn.NET_TIME[sender] += transTime
			}
			peerMap.Add(block.index)
			sendTime := pn.toTimestamp(pn.NET_TIME[sender]+pn.globalLatency) + int64(rand.Intn(1000))
			sendEvent := &SendBlockEvent{
				BaseEvent: BaseEvent{
					timestamp: sendTime,
				},
				block:      block,
				receiverID: p,
			}
			if pn.endTime[block.index] < sendTime {
				pn.endTime[block.index] = sendTime
			}

			result = append(result, sendEvent)
			nextTime = pn.NET_TIME[sender] + pn.globalLatency
			break
		}
	}

	if !allHave {
		if nextTime == -1.0 {
			nextTime = currentTS + 0.1
		}
		checkEvent := &PeerSendEvent{
			BaseEvent: BaseEvent{timestamp: pn.toTimestamp(nextTime)},
			senderID:  sender,
			block:     block,
			network:   pn,
		}
		result = append(result, checkEvent)
	} else {
		if block.minerID != 0 && block.index%50 == 0 && pn.oracle.timestamp == pn.endTime[block.index] {
			log.Warningf("block %d sent to everyone, time %.3f", block.index, float64(pn.endTime[block.index]-pn.startTime[block.index])/pn.oracle.timePrecision)
		}
	}

	return result
}

func (pn *PeerNetwork) expressRelay(block *Block) []Event {
	if pn.attackerIn < 0 {
		return []Event{}
	}
	result := make([]Event, 0)
	for _, attacker := range pn.attacker.List() {
		sendEvent := &SendBlockEvent{
			BaseEvent:  BaseEvent{timestamp: pn.oracle.timestamp + pn.toTimestamp(pn.attackerIn)},
			receiverID: attacker,
			block:      block,
		}
		//log.Criticalf("express block %d", block.index)
		pn.sent[attacker].Add(block.index)
		result = append(result, sendEvent)
	}
	return result
}

func (pn *PeerNetwork) expressBroadcast(block *Block) []Event {
	if pn.attackerOut < 0 {
		return []Event{}
	}
	result := make([]Event, 0)
	for receiver := range pn.sent {
		if receiver == block.minerID {
			continue
		}
		sendEvent := &SendBlockEvent{
			BaseEvent:  BaseEvent{timestamp: pn.oracle.timestamp + pn.toTimestamp(pn.attackerOut)},
			receiverID: receiver,
			block:      block,
		}
		pn.sent[receiver].Add(block.index)
		result = append(result, sendEvent)
	}
	return result
}
