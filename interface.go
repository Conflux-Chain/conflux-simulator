package main

type Miner interface {
	receiveBlock(*Block) ([]*Event)
	generateBlock(*Block) ([]*Event) //The block only need to specify the parent edge and ref edges.
	wake() ([]*Event)
}

type NetworkManager interface {
	getDelay(int, *Block) int64
}

type Event interface {
	getTimestamp() int64
	run(o *Oracle) []*Event
}
