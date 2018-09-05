package main

import (
	"./go-logging"
	"math/rand"
	"time"
	"math"
)

const logLevel = logging.NOTICE
const debug = false

var networkType = BitcoinNet
var attackerR = 0.

const hasAttacker = false

const (
	honestMiners = 10000

	timePrecision = 1e6
	rate          = 5
	blockSize     = 4 // 1MB
	duration      = 600 * rate
)

const (
	//Parameters for Simple Network
	honestDelay = 100
	diameter    = int64(60 * timePrecision)

	//Parameter for Peer Network
	globalLatency = 0.3 // 0.3 second
	bandwidth     = 7.5 //20 Mbps
	peers         = 10

	//Parameters for Simple and Peer with attacker
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
	oracle := NewOracle(timePrecision, rate, duration)
	network := getNetwork(networkType, hasAttacker)

	if hasAttacker {
		attacker := NewHonestMiner()
		//attacker := NewWithholdMiner(delayRef)
		oracle.addMiner(attacker, attackerR/(1-attackerR))
	}

	ratio := 0.8
	sum := (1 - math.Pow(ratio, float64(honestMiners))) / (1 - ratio)
	for i := 0; i < honestMiners; i++ {
		//oracle.addHonestMiner(1 / honestMiners)
		oracle.addHonestMiner(math.Pow(ratio, float64(i)) / sum)
	}

	oracle.setNetwork(network)

	oracle.prepare()
	oracle.run()

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

	log.Error("Start")
	run()
	log.Error("done")
}
