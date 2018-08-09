package main

type EventType int

type eventExecutor func(*Event, *Oracle) ([]*Event)

const (
	mineBlock        EventType = iota + 1
	sendBlock
	broadcastRequest
	wakeNode
)

var executors = map[EventType]eventExecutor{
	mineBlock:        mineBlockExec,
	sendBlock:        sendBlockExec,
	broadcastRequest: broadcastRequestExec,
	wakeNode:         wakeNodeExec}

type sendBlockPayload struct {
	block      *Block
	receiverID int
}

type Event struct {
	timestamp int64
	etype     EventType
	payload   interface{}

	index int
}

func (e *Event) Execute(o *Oracle) {
	results := executors[e.etype](e, o)
	for _, ev := range results {
		if ev.timestamp >= o.timestamp {
			o.queue.Push(ev)
		}
	}
}

func mineBlockExec(e *Event, o *Oracle) ([]*Event) {
	block := e.payload.(*Block)
	miner := *o.getMiner(block.minerID)
	block.seen[block.minerID] = true

	events := miner.generateBlock(block)

	events = append(events, o.mineNextBlock())
	return events
}

func sendBlockExec(e *Event, o *Oracle) ([]*Event) {
	payload := e.payload.(sendBlockPayload)
	receiver := *o.getMiner(payload.receiverID)
	payload.block.seen[payload.receiverID] = true
	return receiver.receiveBlock(payload.block)
}

func broadcastRequestExec(e *Event, o *Oracle) ([]*Event) {
	network := *(o.network)
	block := e.payload.(*Block)

	events := make([]*Event, 0)
	for receiverID, _ := range o.miners.miners {
		if receiverID != block.minerID {
			newevent := &Event{
				timestamp: o.timestamp + network.getDelay(receiverID, block),
				etype:     sendBlock,
				payload:   sendBlockPayload{block: block, receiverID: receiverID},
			}
			events = append(events, newevent)
		}
	}
	return events
}

func wakeNodeExec(e *Event, o *Oracle) ([]*Event) {
	id := e.payload.(int)
	miner := *o.getMiner(id)
	return miner.wake()
}
