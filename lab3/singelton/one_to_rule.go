package main

import (
	ld "da520/lab3/leaderdetector"
	fd "dat20/lab3/failuredetector"
	"flag"
	"fmt"
	"os"
	"time"
)


var (
	help = flag.Bool(
		"help",
		false,
		"Show usage help",
	)
	hostname = flag.String(
		"endpoint",
		"",
		"Endpoint on which server runs or to which client connects",
	)
	port = flag.Int(
		"port",
		0,
		"Port",
	)
	connectto = flag.String(
		"endpoint",
		"",
		"Endpoint for server that will be connected to",
	)
	id = flag.Int(
		"id",
		0,
		"Id of this process",
	)
	nodes = flag.Int(
		"nodes",
		1,
		"Number of nodes in the distributed system",
	)
	delay = flag.Int(
		"delay",
		1000,
		"Delay used by Increasing Timout failuredetector in milliseconds",
	)
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}


func main() {
	flag.Usage = usage
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	nodeIDs := make([]int, *nodes)
	for i := 0; i < *nodes; i++ {
		nodeIDs[i] = i
	}
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	filuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)

	//todo: serve.UDP() function

	
}
