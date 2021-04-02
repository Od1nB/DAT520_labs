package main

import (
	client "dat520/lab5/client/client_struct"
	nt "dat520/lab5/network"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

var (
	help = flag.Bool(
		"help",
		false,
		"Show usage help",
	)
	ip = flag.String(
		"ip",
		"",
		"Ip of the current client.",
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
	debug = flag.Int(
		"debug",
		0,
		"Debug level. Default is 0. 1 for info, 2 for all messages.",
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
	rand.Seed(time.Now().Unix())
	myport := 20000 + *id // + myID
	clientID := fmt.Sprintf("%s:%d", *ip, myport)
	addresses, err := nt.GetServerAddresses(*version, *numNodes, *ports)
	nt.Check(err)
	c := client.NewClient(clientID, *retryLimit, addresses, *debug)

	c.StartClientLoop()
}
