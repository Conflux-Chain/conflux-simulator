package main

import (
	"container/list"
)

type DetailedBlock struct {
	block    *Block
	parent   *DetailedBlock
	maxChild *DetailedBlock
	weight   int
}

func (db *DetailedBlock) isPivot() bool {
	return db.weight <= 0
}

func (db *DetailedBlock) isGenesis() bool {
	return db.block.parent == nil
}

func (db *DetailedBlock) getWeight(g *LocalGraph) float64 {
	if db.isPivot() {
		return float64(g.totalWeight+db.weight) + db.block.residual
	} else {
		return float64(db.weight) + db.block.residual
	}
}

type LocalGraph struct {
	ledger      map[int]*DetailedBlock
	totalWeight int
	tips        *Set
	pivotTip    *DetailedBlock
	genesis      *DetailedBlock
}

func NewLocalGraph() *LocalGraph {
	return &LocalGraph{
		ledger:      make(map[int]*DetailedBlock),
		totalWeight: 0,
		tips:        NewSet(),
		pivotTip:    nil,
	}
}

func (g *LocalGraph) existing(block *Block) bool {
	_, ok := g.ledger[block.index]
	return ok
}

func (g *LocalGraph) getDetailedBlock(block *Block) *DetailedBlock {
	if g.existing(block) {
		return g.ledger[block.index]
	}
	return nil
}

func (g *LocalGraph) seenAllAncestors(block *Block) bool {
	if block.parent != nil && !g.existing(block.parent) {
		//log.Criticalf("don't seen %d",block.parent.index)
		return false
	}
	for _, refBlock := range block.references {
		if !g.existing(refBlock) {
			//log.Criticalf("don't seen %d",refBlock.index)
			return false
		}
	}
	return true
}

func (g *LocalGraph) getAllChildren(db *DetailedBlock) []*DetailedBlock {
	answer := make([]*DetailedBlock, 0)
	for _, childBlock := range db.block.children {
		dChildBlock := g.getDetailedBlock(childBlock)
		if dChildBlock != nil {
			answer = append(answer, dChildBlock)
		}
	}
	return answer
}

func (g *LocalGraph) getAllRefChildren(db *DetailedBlock) []*DetailedBlock {
	answer := make([]*DetailedBlock, 0)
	for _, childBlock := range db.block.refChildren {
		dChildBlock := g.getDetailedBlock(childBlock)
		if dChildBlock != nil {
			answer = append(answer, dChildBlock)
		}
	}
	return answer
}

func (g *LocalGraph) updateTips(db *DetailedBlock) {
	var parents []*Block
	if db.isGenesis() {
		parents = db.block.references
	} else {
		parents = append(db.block.references, db.block.parent)
	}
	for _, refBlock := range parents {
		if g.tips.Has(refBlock.index) {
			g.tips.Remove(refBlock.index)
		}
	}
	g.tips.Add(db.block.index)
}

func (g *LocalGraph) updateMaxChild(db *DetailedBlock) bool {
	oldMaxBlock := db.maxChild
	children := g.getAllChildren(db)

	if len(children) == 0 {
		db.maxChild = nil
		return false
	}

	maxWeight := 0.0
	var maxBlock *DetailedBlock
	for _, childBlock := range children {
		weight := childBlock.getWeight(g)
		if weight > maxWeight {
			maxBlock = childBlock
			maxWeight = weight
		}
	}
	db.maxChild = maxBlock

	if oldMaxBlock == nil || maxBlock.block.index != oldMaxBlock.block.index {
		return true
	} else {
		return false
	}
}

func (g *LocalGraph) fillNewBlock(block *Block) {
	block.parent = g.pivotTip.block
	block.parent.children = append(block.parent.children, block)

	block.height = block.parent.height + 1
	block.ancestorNum = g.totalWeight

	block.references = make([]*Block, 0)
	for _, index := range g.tips.List() {
		if index != block.parent.index {
			refBlock := g.ledger[index].block
			block.references = append(block.references, refBlock)
			refBlock.refChildren = append(refBlock.refChildren, block)
		}
	}
}

type InsertResult int

const (
	Success  InsertResult = iota + 1
	Fail
	Existing
)

func (g *LocalGraph) insert(block *Block) InsertResult {
	if g.existing(block) {
		return Existing
	}

	if !g.seenAllAncestors(block) {
		return Fail
	}

	g.totalWeight = g.totalWeight + 1

	currentBlock := &DetailedBlock{block: block, maxChild: nil, weight: 1}
	if currentBlock.isGenesis() {
		currentBlock.weight = g.totalWeight - currentBlock.weight
		currentBlock.parent = nil
		g.genesis = currentBlock
	} else {
		currentBlock.parent = g.ledger[currentBlock.block.parent.index]
	}
	g.ledger[block.index] = currentBlock

	g.updateTips(currentBlock)

	if currentBlock.isGenesis() {
		g.pivotTip = currentBlock
		return Success
	}

	currentBlock = currentBlock.parent

	for !currentBlock.isPivot() {
		g.updateMaxChild(currentBlock)
		currentBlock.weight = currentBlock.weight + 1
		currentBlock = currentBlock.parent
	}

	pivotPoint := currentBlock
	oldBranch := pivotPoint.maxChild

	currentBlock = oldBranch
	for currentBlock != nil {
		currentBlock.weight = currentBlock.weight - 1
		currentBlock = currentBlock.maxChild
	}

	updated := g.updateMaxChild(pivotPoint)
	newBranch := pivotPoint.maxChild

	if updated {
		currentBlock = oldBranch
		for currentBlock != nil {
			currentBlock.weight = currentBlock.weight + g.totalWeight
			currentBlock = currentBlock.maxChild
		}

		currentBlock = newBranch
		for currentBlock != nil {
			g.pivotTip = currentBlock
			currentBlock.weight = currentBlock.weight - g.totalWeight
			currentBlock = currentBlock.maxChild
		}
	}
	if debug{
		g.checkConsistency()
	}

	return Success
}

/**
 * The following code are used for statistic.
 */

func (g *LocalGraph) getEpochs() map[int]int {
	epochs := make(map[int]int)

	pivotBlock := g.genesis

	epochs[pivotBlock.block.index] = 0
	for pivotBlock.maxChild != nil {
		pivotBlock = pivotBlock.maxChild
		visitList := list.New()
		epoch := pivotBlock.block.height
		visitList.PushBack(pivotBlock)
		for e := visitList.Front(); e != nil; e = e.Next() {
			block := *(e.Value.(*DetailedBlock))
			if _, ok := epochs[block.block.index]; ok {
				continue
			}
			epochs[block.block.index] = epoch
			for _, refblock := range block.block.references {
				visitList.PushBack(g.getDetailedBlock(refblock))
			}
			if !block.isGenesis() {
				visitList.PushBack(block.parent)
			}
		}
	}

	return epochs
}

func (g *LocalGraph) countAnti(c int) (map[int]int, map[int]int) {
	epochMap := g.getEpochs()
	numDesc := make(map[int]int)
	result := make(map[int]int)
	pivotWeight := make(map[int]int)

	for index, epoch := range epochMap {
		if epoch == 0 && index != 0 {
			continue
		}

		visitList := list.New()
		visitedSet := NewSet()
		endEpoch := epoch + c
		count := 0

		visitList.PushBack(g.ledger[index])
		for e := visitList.Front(); e != nil; e = e.Next() {
			block := e.Value.(*DetailedBlock)
			index := block.block.index
			if visitedSet.Has(index) {
				continue
			}
			visitedSet.Add(index)
			if epochMap[index] > endEpoch || (epochMap[index] == 0 && index != 0) {
				continue
			}

			count += 1
			children := g.getAllChildren(block)
			refChildren := g.getAllRefChildren(block)
			allChildren := append(refChildren, children...)
			for _, refblock := range allChildren {
				visitList.PushBack(refblock)
			}
		}
		numDesc[index] = count
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

	for index, descWeight := range numDesc {
		if epochMap[index]+c <= maxEpoch {
			result[index] = pivotWeight[epochMap[index]+c] - (g.ledger[index].block.ancestorNum + descWeight)
			// Debug log
			if result[index] < 0 {
				//	log.Criticalf("block %d, epoch %d, ancestor %d, desc %d, sub graph %d",
				//		g.ledger[index].block.index, epochMap[index], g.ledger[index].block.ancestorNum, descWeight, pivotWeight[epochMap[index]+c])
				//	log.Criticalf("pivot %d", g.pivotTip.block.index)
				//
				//	for id, _ := range make([]int, 15) {
				//		refc := make([]int, len(g.getAllChildren(g.ledger[id])))
				//		for idx, block := range g.getAllChildren(g.ledger[id]) {
				//			refc[idx] = block.block.index
				//		}
				//		refr := make([]int, len(g.getAllRefChildren(g.ledger[id])))
				//		for idx, block := range g.getAllRefChildren(g.ledger[id]) {
				//			refr[idx] = block.block.index
				//		}
				//		log.Criticalf("block %d, epoch %v, refc %v, refr %v", id, epochMap[id], refc, refr)
				//	}
				log.Fatal("")
			}
		}

	}

	//For test only
	//ii := 13
	//if _, ok := result[ii]; ok {
	//	log.Warningf("Block %d have %d + %d in %d, %d anti", ii, g.ledger[ii].block.ancestorNum, numDesc[ii], pivotWeight[epochMap[ii]+c], result[ii])
	//}
	return result, epochMap
}

func (g *LocalGraph) report() {
	weight := g.totalWeight
	pivot := g.pivotTip.block.height
	attackPivot := 0
	attackLastPivot := 0

	pivotBlock := g.pivotTip
	for !pivotBlock.isGenesis() {
		if pivotBlock.block.minerID == 0 {
			attackPivot = attackPivot + 1
			if pivotBlock.block.height+1000 > g.pivotTip.block.height {
				attackLastPivot = attackLastPivot + 1
			}
		}
		pivotBlock = pivotBlock.parent
	}

	log.Warningf("%d(%d) pivot, %d attacker; ratio %.3f, %.3f;",
		pivot, weight, attackPivot, float64(pivot)/float64(weight), float64(attackPivot)/float64(pivot))
}

func (g *LocalGraph) report2(c int) {
	anti, epoch := g.countAnti(c)
	maxEpoch := g.pivotTip.block.height

	ab, aa, hb, ha := 0, 0, 0, 0
	for index, num := range anti {
		if epoch[index] < maxEpoch-100 {
			continue
		}
		if g.ledger[index].block.minerID == 0 {
			ab += 1
			aa += num
		} else {
			hb += 1
			ha += num
		}
	}
	log.Warningf("N+%d Antiset in recent 100 epochs, Attacker %.3f, Honest %.3f", c, float64(aa)/float64(ab), float64(ha)/float64(hb))
}
