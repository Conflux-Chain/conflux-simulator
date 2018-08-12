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

func NewHonestMiner() *HonestMiner {
	return &HonestMiner{
		graph: NewLocalGraph(),
		cache: list.New(),
	}
}

func (hm *HonestMiner) Setup(oracle *Oracle, id int) {
	hm.oracle = oracle
	hm.id = id
}

func (hm *HonestMiner) GenerateBlock(block *Block) []Event {
	// Miners can always seen the genesis block, so block.parent can't be empty
	hm.graph.fillNewBlock(block)
	hm.graph.insert(block)

	// For Log
	refs := make([]int, len(block.references))
	for idx, ref := range block.references {
		refs[idx] = ref.index
	}
	log.Infof("Time %.2f, Miner %d mines block %d, height %d, father %d, refs %v",
			hm.oracle.getRealTime(), hm.id, block.index, block.height, block.parent.index, refs)

	network := hm.oracle.network
	events := network.Broadcast(hm.id, block)

	return events
}

func (hm *HonestMiner) ReceiveBlock(block *Block) []Event {
	network := hm.oracle.network
	events := make([]Event, 0)

	if hm.id == 1 || hm.id == 0 {
		log.Infof("Time %.2f, Miner %d receives %d (miner %d)", hm.oracle.getRealTime(), hm.id, block.index, block.minerID)
	}

	insertResult := hm.graph.insert(block)

	if insertResult == Success {
		results1 := network.Relay(hm.id, block)
		events = append(events, results1...)

		cacheBlocks := hm.insertCache()
		for _, cacheBlock := range cacheBlocks {
			cacheResult := network.Relay(hm.id, cacheBlock)
			events = append(events, cacheResult...)
		}
	} else if insertResult == Fail { // If there are ancestorNum haven't been received, put block to cache.
		hm.cache.PushBack(block)
	}
	return events
}

func (hm *HonestMiner) insertCache() []*Block { // Insert blocks from cache to local graph
	results := make([]*Block, 0)

	if hm.cache.Len() == 0 {
		return results
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
					results = append(results, block)
					updated = true
				}
			}
		}
	}
	return results
}
