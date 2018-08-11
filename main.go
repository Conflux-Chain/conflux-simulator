package main

import (
	"./go-logging"
	_ "time"
	"math/rand"
	"time"
)

const logLevel = logging.WARNING

var attackerR = 0.2

const (
	honestMiners = 256

	timePrecision = 1e6
	rate          = 5
	duration      = 5000

	//Parameter for Peer Network
	blockSize     = 1   // 1MB
	globalLatency = 0.3 // 0.3 second
	bandwidth     = 3   //3 MBps
	peers         = 8

	diameter = int64(60 * timePrecision)

	//Parameters for Simple Network
	honestDelay = 100

	//Parameters for both
	attackerIn  = 2
	attackerOut = 2
)

var log = logging.MustGetLogger("main")

func exp1() *Oracle {
	log.Errorf("Start with attackR %.1f", attackerR)
	oracle := NewOracle(timePrecision, rate, duration)
	network := NewPeerNetwork(false)
	//attacker := NewWithholdMiner(delayRef)
	attacker := NewHonestMiner()
	oracle.addMiner(attacker, attackerR)

	for i := 0; i < honestMiners; i++ {
		oracle.addHonestMiner((1 - attackerR) / honestMiners)
	}

	//oracle.SetSimpleNetwork(true)
	oracle.setNetwork(network)

	log.Notice("done")

	oracle.prepare()
	oracle.run()
	return oracle
}

func main() {
	loadLogger(logLevel)
	//seed := int64(249040116)
	seed := int64(time.Now().Nanosecond())
	rand.Seed(seed)
	log.Noticef("Random seed for this run: %d", seed)

	attackerR = 0
	exp1()
	attackerR = 0.2
	exp1()
	attackerR = 0.3
	exp1()
	attackerR = 0.4
	exp1()

	log.Notice("Done")
}
