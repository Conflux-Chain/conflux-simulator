package main

import (
	"./go-logging"
	"flag"
	"math/rand"
	"time"
)

var (
	logLevel_    int
	debug_       bool
	hasAttacker_ bool
	hasMonopoly_ bool
	attacker_    float64
)

const (
	honestMiners  = 10000
	timePrecision = 1e6
	networkType_  = BitcoinNet
)

var (
	rate_       = 5.0  // 5s
	blockSize_  = 4.0  // 4MB
	bufferSize_ = 16.0 // 16MB socket buffet
	bandwidth_  = 20.0 //20 Mbps
	duration_   = 600 * rate_

	localRatio_ = 0.1 //Parameters for Bitcoin Network
	peers_      = 10
)

const (
	//Parameters for Simple Network
	honestDelay = 100
	diameter    = int64(60 * timePrecision)

	//Parameter for Peer Network
	globalLatency = 0.3 // 0.3 second

	//Parameters for Simple and Peer with attacker
	attackerIn  = 2
	attackerOut = 2
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
	oracle := NewOracle(timePrecision, rate_, duration_)
	network := getNetwork(networkType_, hasAttacker_)

	if hasAttacker_ || hasMonopoly_ {
		attacker := NewHonestMiner()
		//attacker := NewWithholdMiner(delayRef)
		oracle.addMiner(attacker, attacker_/(1-attacker_))
	}

	//ratio := 0.8
	//_ = (1 - math.Pow(ratio, float64(honestMiners))) / (1 - ratio)
	for i := 0; i < honestMiners; i++ {
		oracle.addHonestMiner(1.0 / honestMiners)
		//oracle.addHonestMiner(math.Pow(ratio, float64(i)) / sum)
	}

	oracle.setNetwork(network)

	oracle.prepare()
	oracle.run()

	return oracle
}

func flagParse() {
	flag.BoolVar(&debug_, "d", false, "Set debug")
	flag.IntVar(&logLevel_, "log", 2, "Log Level (1E,2W,3N,4I,5D)")

	flag.Float64Var(&rate_, "r", 5, "Block Generation Rate (s/block)")
	flag.Float64Var(&blockSize_, "s", 4, "Block Size (MB)")
	flag.Float64Var(&bandwidth_, "band", 20, "Bandwidth(Mbps)")
	flag.Float64Var(&bufferSize_, "buff", 32, "Buffer Size (MB)")

	flag.BoolVar(&hasAttacker_, "a", false, "Attacker")
	flag.BoolVar(&hasMonopoly_, "m", false, "Special Honest Miner")
	flag.Float64Var(&attacker_, "l", 0.2, "Attacker ratio")

	flag.Float64Var(&localRatio_, "local", 0.05, "Local ratio")
	flag.IntVar(&peers_, "peer", 10, "Number of peers")

	durblocks := flag.Float64("t", 6000, "Duration (in blocks)")
	flag.Parse()

	duration_ = *durblocks * rate_
	if !hasAttacker_ && !hasMonopoly_ {
		attacker_ = 0
	}
}
func main() {
	flagParse()
	loadLogger(logging.Level(logLevel_))

	log.Warningf("[Running parameters]")
	if !hasAttacker_ && hasMonopoly_ {
		log.Warningf("Basic: rate %0.1f, size %0.0f MB, special honest miner %0.0f%%", rate_, blockSize_, attacker_*100)
	} else {
		log.Warningf("Basic: rate %0.1f, size %0.0f MB, attacker %0.0f%%", rate_, blockSize_, attacker_)
	}
	log.Warningf("Network: bandwidth %0.1f Mbps, %0.1f buffer, %d peers, %d neighbors, local ratio %0.2f", bandwidth_, bufferSize_, honestMiners, peers_, localRatio_)

	var seed int64
	if debug_ {
		seed = int64(499226315)
	} else {
		seed = int64(time.Now().Nanosecond())
	}
	rand.Seed(seed)
	log.Noticef("Random seed for this run: %d", seed)

	log.Error("Start")
	run()
	log.Error("done")
}
