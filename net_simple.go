package main

type SimpleNetwork struct {
	oracle      *Oracle
	honestDelay float64
	attackerIn  float64
	attackerOut float64
	attacker    *Set
}

func NewSimpleNetwork(attacker bool) *SimpleNetwork {
	isAttacker := NewSet()
	if attacker {
		isAttacker.Add(0)
	}
	return &SimpleNetwork{
		honestDelay: honestDelay,
		attackerIn:  attackerIn,
		attackerOut: attackerOut,
		attacker:    isAttacker,
	}
}

func (sn *SimpleNetwork) Setup(oracle *Oracle) {
	sn.oracle = oracle
}

func (sn *SimpleNetwork) Broadcast(senderID int, block *Block) []Event {
	broadcastEvent := &BroadcastEvent{
		BaseEvent: BaseEvent{sn.oracle.timestamp},
		senderID:  senderID,
		block:     block,
		network:   sn,
	}
	return []Event{broadcastEvent}
}

func (sn *SimpleNetwork) Relay(int, *Block) []Event {
	return []Event{}
}

func (sn *SimpleNetwork) getDelay(fromID int, toID int, block *Block) float64 {
	if block.minerID == -1 {
		return 0
	}

	if sn.attacker.Has(block.minerID) {
		return sn.attackerOut
	} else if sn.attacker.Has(toID) {
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

func (e *BroadcastEvent) Run(o *Oracle) ([]Event) {
	log.Debugf("Broadcast Event: time %.2f, block %d, miner %d", o.getRealTime(), e.block.index, e.block.minerID)

	events := make([]Event, 0)
	for receiverID := range o.miners.miners {
		if receiverID != e.block.minerID {
			sendTime := o.timestamp + int64(e.network.getDelay(e.senderID, receiverID, e.block)*o.timePrecision)
			sendEvent := &SendBlockEvent{
				BaseEvent:  BaseEvent{sendTime},
				block:      e.block,
				receiverID: receiverID,
			}
			events = append(events, sendEvent)
		}
	}
	return events
}
