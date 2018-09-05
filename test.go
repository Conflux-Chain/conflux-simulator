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

