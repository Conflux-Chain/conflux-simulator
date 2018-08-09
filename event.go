package main

type BaseEvent struct {
	timestamp int64
}

func (e *BaseEvent) getTimestamp() int64 {
	return e.timestamp
}

type GenBlockEvent struct {
	BaseEvent
	block *Block
}

func (e *GenBlockEvent) run(o *Oracle) []*Event {
	log.Infof("GenBlock Event : time %.2f, block %d, miner %d", o.getTime(), e.block.index, e.block.minerID)

	miner := *o.getMiner(e.block.minerID)
	e.block.seen[e.block.minerID] = true

	if e.block.index%500 == 0 {
		blocks, heights, attack := (*o.miners.miners[1]).(*HonestMiner).graph.report()
		log.Warningf("%d,%d,%d; %.3f, %.3f",
			blocks, heights, attack, float64(heights)/float64(blocks), float64(attack)/float64(heights))
	}
	events := miner.generateBlock(e.block)

	events = append(events, o.mineNextBlock())
	return events
}

type SendBlockEvent struct {
	BaseEvent
	block      *Block
	receiverID int
}

func (e *SendBlockEvent) run(o *Oracle) ([]*Event) {
	log.Infof("SendBlock Event: time %.2f, block %d, receiver %d", o.getTime(), e.block.index, e.receiverID)

	receiver := *o.getMiner(e.receiverID)
	e.block.seen[e.receiverID] = true
	return receiver.receiveBlock(e.block)
}

type BroadcastEvent struct {
	BaseEvent
	senderID int
	block    *Block
}

func (e *BroadcastEvent) run(o *Oracle) ([]*Event) {
	log.Infof("Broadcast Event: time %.2f, block %d, miner %d", o.getTime(), e.block.index, e.block.minerID)

	network := *(o.network)

	events := make([]*Event, 0)
	for receiverID, _ := range o.miners.miners {
		if receiverID != e.block.minerID {
			newevent := new(Event)
			sendTime := o.timestamp + int64(network.getDelay(e.senderID, receiverID, e.block)*o.timePrecision)
			*newevent = &SendBlockEvent{
				BaseEvent:  BaseEvent{sendTime},
				block:      e.block,
				receiverID: receiverID,
			}
			events = append(events, newevent)
		}
	}
	return events
}

type WakeNodeEvent struct {
	BaseEvent
	minerId int
}

func (e *WakeNodeEvent) run(o *Oracle) ([]*Event) {
	miner := *o.getMiner(e.minerId)
	return miner.wake()
}
