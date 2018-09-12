package main

import (
	"math/rand"
	"container/heap"
	"math"
	"container/list"
	"runtime/debug"
)

const logID = 7429

type Traffic struct {
	network    *BitcoinNetwork
	bandwidth  float64
	bufferSize int64
	nodes      map[int]*NodeOutbound
}

func NewTraffic(network *BitcoinNetwork) *Traffic {
	return &Traffic{
		network:    network,
		bandwidth:  bandwidth_,
		nodes:      make(map[int]*NodeOutbound),
		bufferSize: int64(bufferSize_*mb) + 1,
	}
}

type NodeOutbound struct {
	accSize     int64
	lastWakeupT int64
	nextWakeupE *WakeupTrafficEvent
	queue       *PacketEventPriorityQueue
	waiting     *list.List
	buffer      int64
}

func NewNodeOutbound(currentTime int64) *NodeOutbound {
	queue := make(PacketEventPriorityQueue, 0)
	return &NodeOutbound{
		accSize:     0,
		lastWakeupT: currentTime,
		queue:       &queue,
		nextWakeupE: nil,
		waiting:     list.New(),
	}
}

func (t *Traffic) addEvent(e *PacketEvent) []Event {
	currentTime := t.network.oracle.timestamp

	if _, ok := t.nodes[e.senderID]; !ok {
		t.nodes[e.senderID] = NewNodeOutbound(currentTime)
	}
	sender := t.nodes[e.senderID]

	if sender.buffer+e.size < t.bufferSize {
		return t.pushEvent(e)
	} else {
		sender.waiting.PushBack(e)
		return []Event{}
	}
}

func (t *Traffic) pullWaitingListAndUpdate(sender *NodeOutbound) []Event {
	if sender.waiting.Len() == 0 {
		return t.updateNextWakeup(sender)
	}
	front := sender.waiting.Front()
	e := front.Value.(*PacketEvent)
	if sender.buffer+e.size < t.bufferSize {
		sender.waiting.Remove(front)
		return t.pushEvent(e)
	} else {
		return t.updateNextWakeup(sender)
	}
}

func (t *Traffic) pushEvent(e *PacketEvent) []Event {
	oracle := t.network.oracle
	currentTime := t.network.oracle.timestamp

	if _, ok := t.nodes[e.senderID]; !ok {
		t.nodes[e.senderID] = NewNodeOutbound(currentTime)
	}

	sender := t.nodes[e.senderID]

	if sender.queue.Len() > 0 {
		sender.accSize += int64(float64(currentTime-sender.lastWakeupT) * t.bandwidth * 128 * kb / (oracle.timePrecision * float64(sender.queue.Len())))
		sender.lastWakeupT = currentTime
	} else {
		sender.lastWakeupT = currentTime
	}

	e.accSize = sender.accSize + e.size
	sender.buffer += e.size
	e.status = waiting
	heap.Push(sender.queue, e)

	updateE := t.updateNextWakeup(sender)
	return updateE
}

func (t *Traffic) popEvent(e *WakeupTrafficEvent) (*PacketEvent, []Event) {
	sender := e.sender
	currentTime := t.network.oracle.timestamp
	oracle := t.network.oracle

	if sender.queue.Len() == 0 {
		log.Fatal("Sequence Empty")
	}

	sender.accSize += int64(float64(currentTime-sender.lastWakeupT) * t.bandwidth * 128 * kb / (oracle.timePrecision * float64(sender.queue.Len())))
	sender.lastWakeupT = currentTime

	relayEvent := heap.Pop(e.sender.queue).(*PacketEvent)
	sender.buffer -= relayEvent.size

	if math.Abs(float64(relayEvent.accSize-sender.accSize)) > 100*bytes {
		log.Fatalf("%d, %d, %d", relayEvent.senderID, relayEvent.accSize, sender.accSize)
	}
	result := t.pullWaitingListAndUpdate(sender)

	return relayEvent, result
}

func (t *Traffic) updateNextWakeup(sender *NodeOutbound) []Event {
	oracle := t.network.oracle
	currentTime := t.network.oracle.timestamp

	if sender.queue.Len() == 0 {
		sender.nextWakeupE = nil
		return []Event{}
	}

	nextWakeup := currentTime + int64(float64(sender.queue.Peek().accSize-sender.accSize)*oracle.timePrecision*float64(sender.queue.Len())/float64(t.bandwidth*128*kb))
	if nextWakeup < currentTime {
		nextWakeup = currentTime
	}
	if sender.nextWakeupE == nil || sender.nextWakeupE.GetQueue() == nil {
		nextWakeupE := &WakeupTrafficEvent{
			BaseEvent: BaseEvent{timestamp: nextWakeup},
			sender:    sender,
			traffic:   t,
		}
		sender.nextWakeupE = nextWakeupE

		if sender.lastWakeupT != currentTime {
			debug.PrintStack()
			log.Fatal(sender.lastWakeupT, currentTime, "Forget update lastWakeupT")
		}

		if content, ok := t.nodes[logID]; ok && content == sender {
			log.Noticef("Time %0.6f, Newake at %0.6f, %d packets", float64(currentTime)/t.network.oracle.timePrecision, float64(nextWakeup)/t.network.oracle.timePrecision, sender.queue.Len())
		}
		return []Event{nextWakeupE}
	} else {
		sender.nextWakeupE.ChangeTime(nextWakeup)

		if sender.lastWakeupT != currentTime {
			log.Fatal("Forget update lastWakeupT")
		}

		if content, ok := t.nodes[logID]; ok && content == sender {
			log.Noticef("Time %0.6f, Update to %0.6f, %d packets", float64(currentTime)/t.network.oracle.timePrecision, float64(nextWakeup)/t.network.oracle.timePrecision, sender.queue.Len())
		}

		return []Event{}
	}
}

type WakeupTrafficEvent struct {
	BaseEvent
	sender  *NodeOutbound
	traffic *Traffic
}

func (e *WakeupTrafficEvent) Run(oracle *Oracle) []Event {
	t := e.traffic
	currentTime := e.traffic.network.oracle.timestamp

	if content, ok := t.nodes[logID]; ok && content == e.sender {
		log.Noticef("Time %0.6f, Wake up, %d packets remains", float64(currentTime)/t.network.oracle.timePrecision, e.sender.queue.Len()-1)
	}

	trafficE, wakeupR := e.traffic.popEvent(e)

	trafficE.timestamp = currentTime + trafficE.pingDelay()
	trafficE.status = sending

	result := append(wakeupR, trafficE)

	return result
}

type PacketStatus int

const (
	constructed = iota
	waiting
	sending
)

type PacketEvent struct {
	BaseEvent
	network *BitcoinNetwork

	senderID   int
	receiverID int
	size       int64

	accSize      int64
	status       PacketStatus
	childPointer PacketSent //This is a stupid fix. I don't know that golang doesn't have inheritance when I write this code.
}

const (
	bytes = 1000
	kb    = 1024 * bytes
	mb    = 1024 * kb
)

func (e *PacketEvent) pingDelay() int64 {
	network := e.network
	oracle := network.oracle

	loc1, loc2 := network.geo[e.senderID], network.geo[e.receiverID]
	networkDelay := (geodelay[loc1][loc2] + 4*rand.Float64()) * (0.9 + 0.2*rand.Float64())
	return int64(networkDelay / 1000 * oracle.timePrecision)
}

func (e *PacketEvent) prepare(startTime int64) {
	e.timestamp = startTime
	e.status = constructed
}

func (e *PacketEvent) Run(o *Oracle) []Event {
	result := []Event{}
	switch e.status {
	case constructed:
		effects := e.network.traffic.addEvent(e)
		result = append(result, effects...)
	case sending:
		effects := e.childPointer.Sent(o)
		result = append(result, effects...)
	}
	return result
}

func (e *PacketEvent) Sent(o *Oracle) []Event {
	log.Fatal("Not run children implementation")
	return []Event{}
}

//Code for FIFO model

//func (e *PacketEvent) prepare(startTime int64) {
//	network := e.network
//	nextTime := network.nextTime[e.senderID]
//	if nextTime < startTime {
//		nextTime = startTime
//	}
//
//	sentTime := nextTime + e.relayDelay()
//
//	network.nextTime[e.senderID] = sentTime
//	if e.senderID == 0 {
//		log.Debugf("Time %0.2f, miner 0 update finish Time to %0.2f", network.oracle.getRealTime(), float64(sentTime)/network.oracle.timePrecision)
//		if e.size == int(blockSize_*mb) {
//			log.Noticef("Time %0.2f, miner 0 update Time to %0.2f (%0.2fs). (Full block)", network.oracle.getRealTime(), float64(sentTime)/network.oracle.timePrecision, float64(sentTime)/network.oracle.timePrecision-network.oracle.getRealTime())
//		} else if e.size == int(blockSize_*mb/50) {
//			log.Noticef("Time %0.2f, miner 0 update Time to %0.2f (%0.2fs). (Compact block)", network.oracle.getRealTime(), float64(sentTime)/network.oracle.timePrecision, float64(sentTime)/network.oracle.timePrecision-network.oracle.getRealTime())
//		}
//	}
//	e.timestamp = sentTime + e.pingDelay()
//}
//
//func (e *PacketEvent) Run(o *Oracle) []Event {
//	return e.childPointer.Sent(o)
//}
//
//func (e *PacketEvent) relayDelay() int64 {
//	network := e.network
//	oracle := network.oracle
//	traffic := network.traffic
//	realTime := float64(e.size) / (traffic.bandwidth * 128 * kb)
//	return int64(realTime * oracle.timePrecision)
//}
