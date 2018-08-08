package main

type EventType int

type eventExecutor func(*Event, *Oracle) ([]*Event)

const (
	mineBlock        EventType = iota
	sendBlock
	broadcastRequest
	wakeNode
)

var executors = map[EventType]eventExecutor{
	mineBlock:        mineBlockExec,
	sendBlock:        sendBlockExec,
	broadcastRequest: broadcastRequestExec,
	wakeNode:         wakeNodeExec}

type sendBlockPayLoad struct {
	block  *Block
	minerID int
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
	minerId := e.payload.(int)
	miner := *o.getMiner(minerId)
	events, block := miner.generateBlock()

	o.recordNewBlock(minerId, block)
	events = append(events, o.mineNextBlock())
	return events
}

func sendBlockExec(e *Event, o *Oracle) ([]*Event) {
	info := e.payload.(sendBlockPayLoad)
	receiver := *o.getMiner(info.minerID)
	return receiver.receiveBlock(info.block)
}

func wakeNodeExec(e *Event, o *Oracle) ([]*Event) {
	id := e.payload.(int)
	miner := *o.getMiner(id)
	return miner.wake()
}

func broadcastRequestExec(e *Event, o *Oracle) ([]*Event) {
	//TODO: Apply network policy
	return nil
}
