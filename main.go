package main

import "./go-logging"

const loglevel = logging.NOTICE
const timePrecision = 1e5

var log = logging.MustGetLogger("main")

func main() {
	loadLogger(loglevel)

	oracle := NewOracle(timePrecision, 5, 600)

	for i := 0; i < 5; i++ {
		oracle.addHonestMiner(float64(i + 1))
	}
	oracle.setSimpleNetwork(20, 20)

	oracle.prepare()
	oracle.run()

	log.Notice("Done")
}
