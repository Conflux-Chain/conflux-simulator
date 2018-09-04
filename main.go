package main

import (
	"./go-logging"
	"math/rand"
	"time"
)

const logLevel = logging.NOTICE
const debug = false

var networkType = BitcoinNet
var attackerR = 0.

const (
	honestMiners = 10000

	timePrecision = 1e6
	rate          = 5
	duration      = 600 * rate

	//Parameter for Peer Network
	blockSize     = 4   // 1MB
	globalLatency = 0.3 // 0.3 second
	bandwidth     = 7.5 //20 Mbps
	peers         = 10

	//Parameters for Simple Network
	honestDelay = 100
	diameter    = int64(60 * timePrecision)

	//Parameters for Simple and Peer
	attackerIn  = 2
	attackerOut = 2

	//Parameters for Bitcoin Network
	localRatio = 0.1
)

type NetworkType int

const (
	SimpleNet  NetworkType = iota + 1
	PeerNet
	BitcoinNet
)

var log = logging.MustGetLogger("main")

func getNetwork(t NetworkType, attacker bool) Network {
	switch t {
	case SimpleNet:
		return NewSimpleNetwork(attacker)
	case PeerNet:
		return NewPeerNetwork(attacker)
	case BitcoinNet:
		return NewBitcoinNetwork(attacker)
	}
	return nil
}

func run() *Oracle {
	log.Errorf("Start with attackR %.1f", attackerR)

	oracle := NewOracle(timePrecision, rate, duration)
	network := getNetwork(networkType, attackerR <= 0)

	if attackerR > 0 {
		attacker := NewHonestMiner()
		//attacker := NewWithholdMiner(delayRef)
		oracle.addMiner(attacker, attackerR)
	}

	for i := 0; i < honestMiners; i++ {
		oracle.addHonestMiner((1 - attackerR) / honestMiners)
	}

	oracle.setNetwork(network)

	oracle.prepare()
	oracle.run()

	log.Error("done")
	return oracle
}

func main() {
	loadLogger(logLevel)
	var seed int64
	if debug {
		seed = int64(249020116)
	} else {
		seed = int64(time.Now().Nanosecond())
	}
	rand.Seed(seed)
	log.Noticef("Random seed for this run: %d", seed)

	run()
}
