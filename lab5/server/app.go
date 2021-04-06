package main

import (
	nt "dat520/lab5/network"
	server "dat520/lab5/server/server_struct"
	"flag"
	"fmt"
	"os"
)

var (
	help = flag.Bool(
		"help",
		false,
		"Show usage help",
	)
	id = flag.Int(
		"id",
		0,
		"Id of the current client.",
	)
	ports = flag.Int(
		"ports",
		19000,
		"Ports for servers.",
	)
	numNodes = flag.Int(
		"n",
		3,
		"Number of servers.",
	)
	retryLimit = flag.Int(
		"retry",
		0,
		"Id of the current client.",
	)
	version = flag.Int(
		"v",
		0,
		"0 for localhost, 1 for uis unix and 2 for docker.",
	)
	delay = flag.Int(
		"delay",
		1000,
		"Delay used by Increasing Timout failuredetector in milliseconds",
	)
	debug = flag.Int(
		"debug",
		0,
		"Debug level. Default is 0. 1 for info, 2 for all paxos messages except heartbeat, 3 for all messages.",
	)
	passive = flag.Bool(
		"passive",
		false,
		"Boolean for setting up server without it taking part of mpaxos, false for not passive and true for it being passive",
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
	addresses, err := nt.GetServerAddresses(*version, 7, *ports)
	nt.Check(err)
	s := server.NewServer(*id, *delay, *retryLimit, addresses,*numNodes, *debug)
	if *passive{
		//make a dummy server that only waits for reconfig message to arrive
		//make function in server struct that takes care of this
	} else{
		s.StartServerLoop()
	}
}
