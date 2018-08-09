package main

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
	tips        map[int]bool
	pivotTip    *DetailedBlock
}

func (g *LocalGraph) existing(block *Block) bool {
	_, ok := g.ledger[block.index]
	return ok
}

func (g *LocalGraph) seenAllAncestors(block *Block) bool {
	if block.parent != nil && !g.existing(block.parent) {
		return false
	}
	for _, refBlock := range block.references {
		if !g.existing(refBlock) {
			return false
		}
	}
	return true
}

func (g *LocalGraph) getAllChildren(db *DetailedBlock) []*DetailedBlock {
	answer := make([]*DetailedBlock, 0)
	for _, childBlock := range db.block.children {
		dChildBlock, ok := g.ledger[childBlock.index]
		if ok {
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
		if _, ok := g.tips[refBlock.index]; ok {
			delete(g.tips, refBlock.index)
		}
	}
	g.tips[db.block.index] = true
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

func (g *LocalGraph) insert(block *Block) bool {
	if g.existing(block) {
		return true
	}

	if !g.seenAllAncestors(block) {
		return false
	}

	g.totalWeight = g.totalWeight + 1

	currentBlock := &DetailedBlock{block: block, maxChild: nil, weight: 1}
	if currentBlock.isGenesis() {
		currentBlock.weight = g.totalWeight - currentBlock.weight
		currentBlock.parent = nil
	} else {
		currentBlock.parent = g.ledger[currentBlock.block.parent.index]
	}
	g.ledger[block.index] = currentBlock

	g.updateTips(currentBlock)

	if currentBlock.isGenesis() {
		g.pivotTip = currentBlock
		return true
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
	g.checkConsistency()
	return true
}