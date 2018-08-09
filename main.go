package main

import (
	"./go-logging"
	"time"
	"math/rand"
)

const loglevel = logging.WARNING
const timePrecision = 1e5

var log = logging.MustGetLogger("main")

func exp1() *Oracle {
	oracle := NewOracle(timePrecision, 5, 864000)

	oracle.addHonestMiner(0.2)
	for i := 0; i < 30; i++ {
		oracle.addHonestMiner(0.8 / 30)
	}

	network := new(NetworkManager)
	*network = &SimpleNetwork{
		honestDelay: 100,
		attackerIn:  5,
		attackerOut: 5,
		isAttacker:  map[int]bool{0:true},
	}
	oracle.network = network

	oracle.prepare()
	oracle.run()
	return oracle
}

func main() {
	loadLogger(loglevel)
	seed := int64(time.Now().Nanosecond())
	rand.Seed(seed)
	log.Noticef("Random seed for this run: %d", seed)

	exp1()

	log.Notice("Done")
}
