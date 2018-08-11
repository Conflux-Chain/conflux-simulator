package main

import "container/list"

type WMinerType int

const (
	selfish  WMinerType = iota + 1
	delayRef
)

type WithholdMiner struct {
	id     int
	mType  WMinerType
	oracle *Oracle
	graph  *LocalGraph
	cache  *list.List

	realGraph     *LocalGraph
	holdingBlock  *list.List
	receivingTime map[int]int64
}

func NewWithholdMiner(t WMinerType) *WithholdMiner {
	return &WithholdMiner{
		mType:         t,
		graph:         NewLocalGraph(),
		cache:         list.New(),
		realGraph:     NewLocalGraph(),
		holdingBlock:  list.New(),
		receivingTime: make(map[int]int64),
	}
}

func (wm *WithholdMiner) Setup(oracle *Oracle, id int) {
	wm.oracle = oracle
	wm.id = id
}

func (wm *WithholdMiner) GenerateBlock(block *Block) []Event {
	// Miners can always seen the gensis block, so block.parent can't be empty
	if wm.mType == selfish {
		block.parent = wm.graph.pivotTip.block
		block.parent.children = append(block.parent.children, block)

		block.height = block.parent.height + 1
		block.ancestorNum = block.parent.ancestorNum + 1
	} else if wm.mType == delayRef {
		wm.graph.fillNewBlock(block)
	}

	wm.graph.insert(block)

	refs := make([]int, len(block.references))
	for idx, ref := range block.references {
		refs[idx] = ref.index
	}

	log.Noticef("Time %.2f, Miner %d mines %d, height %d, father %d, refs %v",
		wm.oracle.getRealTime(), wm.id, block.index, block.height, block.parent.index, refs)

	wm.holdingBlock.PushBack(block)
	return wm.checkBroadCast()
}

func (wm *WithholdMiner) ReceiveBlock(block *Block) []Event {
	wm.receivingTime[block.index] = wm.oracle.timestamp
	if block.minerID == -1 {
		wm.realGraph.insert(block)
		wm.graph.insert(block)
		return []Event{}
	}

	if wm.id == 0 {
		log.Infof("Time %.2f, Miner %d receives %d", wm.oracle.getRealTime(), wm.id, block.index)
	}

	insertResult := wm.realGraph.insert(block)

	switch insertResult {
	case Fail:
		log.Fatalf("block %d", block.index)
		wm.cache.PushBack(block)
		return []Event{}
	case Existing:
		return []Event{}
	case Success:
		result0 := wm.graphInsert(block)
		result1 := wm.insertCache()
		result2 := wm.checkBroadCast()
		result := append(result0, result1...)
		result = append(result, result2...)
		return result

	}
	return []Event{}
}

func (wm *WithholdMiner) graphInsert(block *Block) []Event {
	switch wm.mType {
	case selfish:
		wm.graph.insert(block)
		return []Event{}
	case delayRef:
		if wm.receivingTime[block.index]+diameter/2 <= wm.oracle.timestamp {
			wm.graph.insert(block)
			return []Event{}
		} else {
			delayEvent := &DelayInsertEvent{
				BaseEvent: BaseEvent{wm.oracle.timestamp + diameter/2},
				block:     block,
				m:         wm,
			}
			return []Event{delayEvent}
		}

	}
	return []Event{}
}

func (wm *WithholdMiner) checkBroadCast() []Event {
	network := wm.oracle.network
	events := make([]Event, 0)
	for wm.realGraph.pivotTip.block.minerID != wm.id && wm.holdingBlock.Len() > 0 {
		e := wm.holdingBlock.Front()
		broadcastBlock := e.Value.(*Block)
		wm.holdingBlock.Remove(e)
		result := network.Broadcast(wm.id, broadcastBlock)
		events = append(events, result...)
		wm.realGraph.insert(broadcastBlock)
		log.Noticef("Time %.2f, AdvMiner broadcast %d",
			wm.oracle.getRealTime(), broadcastBlock.index)
	}
	return events
}

func (wm *WithholdMiner) insertCache() []Event { // Insert blocks from cache to local graph
	if wm.cache.Len() == 0 {
		return []Event{}
	}
	updated := true
	results := []Event{}
	for updated {
		updated = false
		for e := wm.cache.Front(); e != nil; e = e.Next() {
			block := e.Value.(*Block)
			insertResult := wm.realGraph.insert(block)
			if insertResult != Fail {
				wm.cache.Remove(e)
				if insertResult == Success {
					results = append(results, wm.graphInsert(block)...)
					updated = true
				}
			}
		}
	}
	return results
}

type DelayInsertEvent struct {
	BaseEvent
	m     *WithholdMiner
	block *Block
}

func (e *DelayInsertEvent) Run(o *Oracle) []Event {
	e.m.graph.insert(e.block)
	return []Event{}
}
