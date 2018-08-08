package main

type Miner interface {
	setId(int)
	receiveBlock(*Block) ([]*Event)
	generateBlock() ([]*Event, *Block) //The block only need to specify the parent edge and ref edges.
	wake() ([]*Event)
}
