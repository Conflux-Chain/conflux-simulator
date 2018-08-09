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
		tips:        make(map[int]bool),
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

func (hm *HonestMiner) generateBlock(block *Block) []*Event {
	// Miners can always seen the gensis block, so block.parent can't be empty

	block.parent = hm.graph.pivotTip.block
	block.height = block.parent.height + 1
	block.ancestors = hm.graph.totalWeight

	block.references = make([]*Block, 0)
	for index, _ := range hm.graph.tips {
		if index != block.parent.index {
			block.references = append(block.references, hm.graph.ledger[index].block)
		}
	}
	block.parent.children = append(block.parent.children, block)

	log.Noticef("Time %.2f, Miner %d mines %d, height %d, father %d",
		hm.oracle.getTime(), hm.id, block.index, block.height, block.parent.index)

	hm.graph.insert(block)

	broadcastEvent := new(Event)
	*broadcastEvent = &BroadcastEvent{
		BaseEvent: BaseEvent{hm.oracle.timestamp},
		block:     block,
	}
	return []*Event{broadcastEvent}
}

func (hm *HonestMiner) receiveBlock(block *Block) []*Event {
	if hm.graph.insert(block) {
		hm.insertCache()
	} else {
		hm.cache.PushBack(block) // If there are ancestors haven't been received, put block to cache.
	}
	return []*Event{}
}

func (hm *HonestMiner) wake() []*Event {
	return []*Event{}
}

func (hm *HonestMiner) insertCache() { // Insert blocks from cache to local graph
	if hm.cache.Len() == 0 {
		return
	}
	updated := true
	for updated {
		updated = false
		for e := hm.cache.Front(); e != nil; e = e.Next() {
			success := hm.graph.insert(e.Value.(*Block))
			if success {
				hm.cache.Remove(e)
				updated = true
			}
		}
	}
}
