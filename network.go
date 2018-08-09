package main

type SimpleNetwork struct {
	honestDelay   int64
	attackerDelay int64
	isAttacker    map[int]bool
}

func (sn *SimpleNetwork) getDelay(toID int, block *Block) int64 {
	if block.minerID == 0 {
		return 0
	}

	_, attks := sn.isAttacker[block.minerID]
	_, attkr := sn.isAttacker[toID]

	if attks || attkr {
		return sn.attackerDelay
	} else {
		return sn.honestDelay
	}
}
