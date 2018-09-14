package main

import "container/heap"

type Miner interface {
	Setup(*Oracle, int)
	ReceiveBlock(*Block) ([]Event)
	GenerateBlock(*Block) ([]Event) //The block only need to specify the parent edge and ref edges.
}

type Network interface {
	Setup(*Oracle)
	Broadcast(int, *Block) []Event
	Relay(int, *Block) []Event
}

type Event interface {
	GetTimestamp() int64
	Run(o *Oracle) []Event
	SetIndex(int)
	GetIndex() int
	SetQueue(*EventQueue)
}

type PacketSent interface {
	Sent(o *Oracle) []Event //Called when the send have sent all the information
}

type BaseEvent struct {
	timestamp int64
	index     int
	eq        *EventQueue
}

func (e *BaseEvent) GetTimestamp() int64 {
	return e.timestamp
}

func (e *BaseEvent) SetIndex(id int) {
	e.index = id
}

func (e *BaseEvent) GetIndex() int {
	return e.index
}

func (e *BaseEvent) SetQueue(eq *EventQueue) {
	e.eq = eq
}

func (e *BaseEvent) GetQueue() *EventQueue {
	return e.eq
}

func (e *BaseEvent) ChangeTime(timestamp int64) {
	e.timestamp = timestamp
	if e.eq == nil {
		return
	} else {
		heap.Fix(e.eq.queueList, e.index)
	}
}
