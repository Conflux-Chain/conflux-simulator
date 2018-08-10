package main

type PeerNetwork struct {
	oracle *Oracle

	sent     map[int]*Set
	peer     map[int]([]int)
	NET_TIME []float64

	blockSize     float64
	bandwidth     float64
	globalLatency float64
}

func (pn *PeerNetwork) toTimestamp(t float64, ) int64 {
	return int64(t * float64(pn.oracle.timePrecision))
}

func (pn *PeerNetwork) Relay(id int, block *Block) []*Event {
	sendToPeer := new(Event)
	*sendToPeer = &PeerSendEvent{
		BaseEvent: BaseEvent{pn.oracle.timestamp},
		senderID:  id,
		block:     block,
		network:   pn,
	}
	return []*Event{sendToPeer}
}

func (pn *PeerNetwork) Broadcast(id int, block *Block) []*Event {
	return pn.Relay(id, block)
}

func (pn *PeerNetwork) sendBlockToBestPeer(e *PeerSendEvent) []*Event {
	result := []*Event{}
	sender := e.senderID
	block := e.block
	currentTS := pn.oracle.getRealTime()

	allHave := true
	nextTime := -1.0

	for _, p := range pn.peer[sender] {
		peerMap := pn.sent[p]
		if !peerMap.Has(block.index) {
			allHave = false
			transTime := pn.blockSize * 8 / pn.bandwidth * 1.0
			if pn.NET_TIME[sender] <= currentTS {
				pn.NET_TIME[sender] = currentTS + transTime
			} else if pn.NET_TIME[sender]-currentTS > 5.0 {
				return result
			} else {
				pn.NET_TIME[sender] += transTime
			}
			peerMap.Add(block.index)
			sendEvent := new(Event)
			*sendEvent = &SendBlockEvent{
				BaseEvent: BaseEvent{
					timestamp: pn.toTimestamp(pn.NET_TIME[sender] + pn.globalLatency),
				},
				block:      block,
				receiverID: p,
			}
			result = append(result, sendEvent)
			nextTime = pn.NET_TIME[sender] + pn.globalLatency
			break
		}
	}

	if !allHave {
		if nextTime == -1.0 {
			nextTime = currentTS + 0.1
		}
		checkEvent := new(Event)
		*checkEvent = &PeerSendEvent{
			BaseEvent: BaseEvent{pn.toTimestamp(nextTime)},
			senderID:  sender,
			block:     block,
			network:   pn,
		}
		result = append(result, checkEvent)
	}

	return result
}

type PeerSendEvent struct {
	BaseEvent
	senderID int
	block    *Block
	network  *PeerNetwork
}

func (e *PeerSendEvent) Run(o *Oracle) ([]*Event) {
	log.Infof("PeerSend  Event: time %.2f, block %d, sender %d", o.getRealTime(), e.block.index, e.senderID)
	return e.network.sendBlockToBestPeer(e)
}
