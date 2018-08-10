package main

import (
	"./go-logging"
	"time"
	"math/rand"
)

const loglevel = logging.WARNING
const (
	timePrecision = 1e5
	rate          = 5
	duration      = 864000

	//Parameter for Peer Network
	blockSize     = 1   // 1MB
	globalLatency = 0.3 // 0.3 second
	bandwidth     = 20  //20 MBps
	peers         = 16

	//Paremeter for Simple Network
	honestDelay = 100
	attackerIn  = 5
	attackerOut = 5
)

var log = logging.MustGetLogger("main")

func exp1() *Oracle {
	oracle := NewOracle(timePrecision, rate, duration)

	oracle.addHonestMiner(0.2)
	for i := 0; i < 250; i++ {
		oracle.addHonestMiner(0.8 / 250)
	}

	//oracle.SetSimpleNetwork(true)
	oracle.SetPeerNetwork()
	log.Notice("done")

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
