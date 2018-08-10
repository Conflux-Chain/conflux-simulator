package main

type SimpleNetwork struct {
	oracle      *Oracle
	honestDelay float64
	attackerIn  float64
	attackerOut float64
	isAttacker  *Set
}

func (sn *SimpleNetwork) Relay(int, *Block) []*Event {
	return []*Event{}
}

func (sn *SimpleNetwork) Broadcast(senderID int, block *Block) []*Event {
	broadcastEvent := new(Event)
	*broadcastEvent = &BroadcastEvent{
		BaseEvent: BaseEvent{sn.oracle.timestamp},
		senderID:  senderID,
		block:     block,
		network:   sn,
	}
	return []*Event{broadcastEvent}
}

func (sn *SimpleNetwork) getDelay(fromID int, toID int, block *Block) float64 {
	if block.minerID == -1 {
		return 0
	}

	if sn.isAttacker.Has(block.minerID) {
		return sn.attackerOut
	} else if sn.isAttacker.Has(toID) {
		return sn.attackerIn
	} else {
		return sn.honestDelay
	}
}

type BroadcastEvent struct {
	BaseEvent
	senderID int
	block    *Block
	network  *SimpleNetwork
}

func (e *BroadcastEvent) Run(o *Oracle) ([]*Event) {
	log.Infof("Broadcast Event: time %.2f, block %d, miner %d", o.getRealTime(), e.block.index, e.block.minerID)

	events := make([]*Event, 0)
	for receiverID, _ := range o.miners.miners {
		if receiverID != e.block.minerID {
			newevent := new(Event)
			sendTime := o.timestamp + int64(e.network.getDelay(e.senderID, receiverID, e.block)*o.timePrecision)
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
