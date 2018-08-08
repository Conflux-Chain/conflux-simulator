package main

type Miner interface {
	setId(int)
	receiveBlock(*Block) ([]*Event)
	generateBlock(*Block) ([]*Event) //The block only need to specify the parent edge and ref edges.
	wake() ([]*Event)
}



type HonestMiner struct {

}