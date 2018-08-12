package main

// This file contains function for correctness check. You don't need to read the details.

func (g *LocalGraph) checkConsistency() {
	count := 0
	count2 := 1

	tips := make(map[int]bool)
	if g.ledger[0].weight != 0 {
		log.Fatal("local graph error: genesis error")
	}
	for id, block := range g.ledger {
		count = count + 1
		if id != block.block.index {
			log.Fatal("local graph error: index consistency")
		}

		_, havetip := tips[block.block.index]
		if !havetip {
			tips[block.block.index] = false
		}

		children := g.getAllChildren(block)
		totalWeight := 1
		maxWeight := 0.0
		var maxblock *DetailedBlock
		maxblock = nil

		for _, tip := range block.block.references {
			tips[tip.index] = true
		}
		if !block.isGenesis() {
			tips[block.block.parent.index] = true
		}

		for _, child := range children {
			count2 = count2 + 1
			if child.parent != block {
				log.Fatal("local graph error: child parent consistency")
			}

			if child.isPivot() {
				totalWeight = totalWeight + child.weight + g.totalWeight
				if !block.isPivot() || child != block.maxChild {
					log.Fatal("local graph error: mark non-pivot block as pivot")
				}
			} else {
				totalWeight = totalWeight + child.weight
				if block.isPivot() && child == block.maxChild {
					log.Fatal("local graph error: mark pivot block as non-pivot")
				}
			}

			if maxWeight < child.getWeight(g) {
				maxWeight = child.getWeight(g)
				maxblock = child
			}
		}
		if block.maxChild != maxblock {
			log.Fatalf("local graph error: max child consistency, say %v, find %v", block.maxChild, maxblock)
		}
		if block.isPivot() && totalWeight != block.weight+g.totalWeight {
			log.Fatalf("block %d, local graph error: weight consistency", block.block.index)
		}
		if !block.isPivot() && totalWeight != block.weight {
			log.Fatal("local graph error: weight consistency")
		}
	}
	if count != g.totalWeight || count != count2 {
		log.Fatal("local graph error: global weight consistency")
	}
	if !g.pivotTip.isPivot() || g.pivotTip.maxChild != nil {
		log.Fatal("local graph error: pivot tip error")
	}
	for idx, having := range tips {
		if !g.tips.Has(idx) && !having {
			log.Fatalf("local graph error: find tip block %d outside tip list", idx)
		}
		if g.tips.Has(idx) && having {
			log.Fatal("local graph error: find non-tip block in tip list")
		}
	}
}

// A wrong implementation

type VisitBlock struct {
	index     int
	visitList []*DetailedBlock
	nextVisit int
	weights   *CountMap
	epoch     int
}

func NewVisitBlock(block *DetailedBlock, g *LocalGraph, epoch int) *VisitBlock {
	return &VisitBlock{
		index:     block.block.index,
		visitList: g.getAllChildren(block),
		nextVisit: 0,
		weights:   NewCountMap(),
		epoch:     epoch,
	}
}

func (g *LocalGraph) __countLimitAnticone(c int) map[int]int {
	epochs := g.getEpochs()
	limitDesc := make(map[int]int)
	pivotWeight := make(map[int]int)
	result := make(map[int]int)

	visitStack := NewStack()
	visitStack.Push(NewVisitBlock(g.genesis, g, 0))

	for {
		v := visitStack.Peek().(*VisitBlock)
		if v.nextVisit >= len(v.visitList) {
			v.weights.Incur(v.epoch, 1)
			limitDesc[v.index] = v.weights.Sum()
			if v.epoch == 0 {
				//for int, num := range *v.weights{
				//	log.Notice(int,num)
				//}
				log.Noticef("run block %d", v.index)
			}
			visitStack.Pop()
			if visitStack.Len() == 0 {
				break
			}
			parent := visitStack.Peek().(*VisitBlock)

			parent.weights.Merge(v.weights)
			for epoch := v.epoch; epoch > parent.epoch; epoch -= 1 {
				parent.weights.Remove(epoch + c)
			}
			parent.nextVisit += 1
		} else {
			nextBlock := v.visitList[v.nextVisit]
			nextEpoch := epochs[nextBlock.block.index]
			if nextEpoch == 0 && !nextBlock.isGenesis() {
				v.nextVisit += 1
			} else {
				visitStack.Push(NewVisitBlock(nextBlock, g, nextEpoch))
			}
		}
	}

	pivotBlock := g.genesis
	pivotWeight[0] = 1
	epoch := 0

	for pivotBlock.maxChild != nil {
		pivotBlock = pivotBlock.maxChild
		epoch = epoch + 1
		pivotWeight[epoch] = pivotBlock.block.ancestorNum + 1
	}
	maxEpoch := epoch

	for index, descWeight := range limitDesc {
		if epochs[index]+c <= maxEpoch {
			result[index] = pivotWeight[epochs[index]+c] - (g.ledger[index].block.ancestorNum + descWeight)
		}
	}
	ii := 1003
	if _, ok := limitDesc[ii]; ok {
		log.Warningf("Genesis block have %d + %d in %d, %d anti", g.ledger[ii].block.ancestorNum, limitDesc[ii], pivotWeight[epochs[ii]+c], result[ii])

	}

	return result

}
