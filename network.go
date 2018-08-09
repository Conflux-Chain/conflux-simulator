package main

type SimpleNetwork struct {
	honestDelay float64
	attackerIn  float64
	attackerOut float64
	isAttacker  map[int]bool
}

func (sn *SimpleNetwork) getDelay(toID int, block *Block) float64 {
	if block.minerID == -1 {
		return 0
	}

	_, attks := sn.isAttacker[block.minerID]
	_, attkr := sn.isAttacker[toID]

	if attks {
		return sn.attackerOut
	} else if attkr {
		return sn.attackerIn
	} else {
		return sn.honestDelay
	}
}
