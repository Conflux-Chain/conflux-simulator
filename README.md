#Conflux Simulator

##Intro

In `conflux_simulation_pool.go`, each mining pool is only represented by one node.
In `conflux_simulation_net.go`, we are considering adversaries with network advantages. The main difference is in the function `procEvent`.

##Running the Simulator

Simply execute `go run conflux_simulation_pool.go`.