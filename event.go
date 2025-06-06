package main

import "time"

type GenBlockEvent struct {
	BaseEvent
	block *Block
}

func (e *GenBlockEvent) Run(o *Oracle) []Event {
	log.Debugf("GenBlock  Event: time %.2f, block %d, miner %d", o.getRealTime(), e.block.index, e.block.minerID)

	miner := o.getMiner(e.block.minerID)
	e.block.seen[e.block.minerID] = true

	if e.block.index%50 == 0 {
		t := float64(o.timestamp) / o.timePrecision
		log.Warning("")
		log.Warningf("Current time: %.2f s", t)

		viewGraph := o.miners.miners[1].(*HonestMiner).graph

		log.Noticef("Pivot block %d", viewGraph.pivotTip.block.index)
		viewGraph.report_pivot()

		if e.block.index%50 == 0 {
			viewGraph.report_anti(20)
			viewGraph.report_epochsize()
		}

		time.Sleep(1 * time.Millisecond)
		log.Warning("")
	}
	//time.Sleep(300 * time.Millisecond)
	events := miner.GenerateBlock(e.block)

	events = append(events, o.mineNextBlock())
	return events
}

type SendBlockEvent struct {
	BaseEvent
	block      *Block
	receiverID int
}

func (e *SendBlockEvent) Run(o *Oracle) ([]Event) {
	log.Debugf("SendBlock Event: time %.2f, block %d, receiver %d", o.getRealTime(), e.block.index, e.receiverID)

	receiver := o.getMiner(e.receiverID)
	e.block.seen[e.receiverID] = true
	return receiver.ReceiveBlock(e.block)
}
