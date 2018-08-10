package main

import (
	"container/list"
)

type HonestMiner struct {
	id     int
	oracle *Oracle
	graph  *LocalGraph
	cache  *list.List
}

func NewHonestMiner(id int, oracle *Oracle) *HonestMiner {
	graph := &LocalGraph{
		ledger:      make(map[int]*DetailedBlock),
		totalWeight: 0,
		tips:        NewSet(),
		pivotTip:    nil,
	}
	cache := list.New()
	return &HonestMiner{
		id:     id,
		oracle: oracle,
		graph:  graph,
		cache:  cache,
	}
}

func (hm *HonestMiner) GenerateBlock(block *Block) []*Event {
	// Miners can always seen the gensis block, so block.parent can't be empty
	hm.graph.fillNewBlock(block)
	hm.graph.insert(block)

	log.Noticef("Time %.2f, Miner %d mines %d, height %d, father %d",
		hm.oracle.getRealTime(), hm.id, block.index, block.height, block.parent.index)

	network := *hm.oracle.network
	events := network.Broadcast(hm.id, block)

	return events
}

func (hm *HonestMiner) ReceiveBlock(block *Block) []*Event {
	network := *hm.oracle.network
	events := make([]*Event, 0)
	insertResult := hm.graph.insert(block)
	if insertResult == Success {
		results1 := network.Relay(hm.id, block)
		results2 := hm.insertCache()
		events = append(results1, results2...)
	} else if insertResult == Fail { // If there are ancestorNum haven't been received, put block to cache.
		hm.cache.PushBack(block)
	}
	return events
}

func (hm *HonestMiner) insertCache() []*Event { // Insert blocks from cache to local graph
	events := make([]*Event, 0)
	network := *hm.oracle.network

	if hm.cache.Len() == 0 {
		return []*Event{}
	}
	updated := true
	for updated {
		updated = false
		for e := hm.cache.Front(); e != nil; e = e.Next() {
			block := e.Value.(*Block)
			insertResult := hm.graph.insert(block)
			if insertResult != Fail {
				hm.cache.Remove(e)
				if insertResult == Success {
					results := network.Relay(hm.id, block)
					events = append(events, results...)
					updated = true
				}
			}
		}
	}
	return events
}
