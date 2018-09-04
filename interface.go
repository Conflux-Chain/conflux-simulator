package main

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
}

type BaseEvent struct {
	timestamp int64
}

func (e *BaseEvent) GetTimestamp() int64 {
	return e.timestamp
}