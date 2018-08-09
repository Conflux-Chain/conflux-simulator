package main

type Miner interface {
	setup(int, *Oracle)
	receiveBlock(*Block) ([]*Event)
	generateBlock(*Block) ([]*Event) //The block only need to specify the parent edge and ref edges.
	wake() ([]*Event)
}

type NetworkManager interface {
	getDelay(int, *Block) int64
}
