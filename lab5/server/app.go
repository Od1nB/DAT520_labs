package main

import (
	nt "dat520/lab5/network"
	server "dat520/lab5/server/server_struct"
	"flag"
	"fmt"
	"net"
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
	var addresses []*net.UDPAddr
	var addr *net.UDPAddr
	var err error
	if *numNodes == 1 {
		addr, err = net.ResolveUDPAddr("udp", "192.168.1.4:19000")
		addresses = []*net.UDPAddr{addr}
	} else {
		addresses, err = nt.GetServerAddresses(*version, *numNodes, *ports)
		nt.Check(err)
	}
	s := server.NewServer(*id, *delay, *retryLimit, addresses, *numNodes, *debug)
	s.StartServerLoop()
}
