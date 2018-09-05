package main

import (
	"./go-logging"
	"math/rand"
	"time"
	"math"
	"flag"
)

var (
	logLevel_    int
	debug_       bool
	networkType_ NetworkType
	hasAttacker_ bool
	attacker_    float64
)

const (
	honestMiners = 10000

	timePrecision = 1e6
	rate          = 5.0
	blockSize     = 4.0 // 1MB
)

var duration_ = 600 * rate

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

var (
	localRatio_ = 0.1 //Parameters for Bitcoin Network
	bandwidth_  = 7.5 //20 Mbps
	peers_      = 10
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
	oracle := NewOracle(timePrecision, rate, duration_)
	network := getNetwork(networkType_, hasAttacker_)

	if hasAttacker_ {
		attacker := NewHonestMiner()
		//attacker := NewWithholdMiner(delayRef)
		oracle.addMiner(attacker, attacker_/(1-attacker_))
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

func flagParse() {
	debug_ = *flag.Bool("d", false, "debug")
	hasAttacker_ = *flag.Bool("a", false, "attacker")
	logLevel_ = *flag.Int("l", 3, "Log Level")
	attacker_ = *flag.Float64("r", 0.2, "Attacker ratio")
	localRatio_ = *flag.Float64("local", 0.2, "Local ratio")
	bandwidth_ = *flag.Float64("band", 7.5, "Bandwidth(Mbps)")
	peers_ = *flag.Int("peer", 10, "Number of peers")
	duration_ = rate * *flag.Float64("t", 600, "Duration (in blocks)")

	networkType_ = NetworkType(*flag.Int("net", 3, "Attacker ratio"))
	flag.Parse()
}
func main() {
	flagParse()
	loadLogger(logging.Level(logLevel_))

	var seed int64
	if debug_ {
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
