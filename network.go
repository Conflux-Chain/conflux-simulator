package main

type NetworkManager interface {
	getDelay(int, *Block) int64
}

type SimpleNetwork struct {
	honestDelay   int64
	attackerDelay int64
	isAttacker    map[int]bool
}

func (sn *SimpleNetwork) getDelay(toID int, block *Block) int64 {
	if sn.isAttacker[block.minerID] || sn.isAttacker[toID] {
		return sn.attackerDelay
	} else {
		return sn.honestDelay
	}
}
