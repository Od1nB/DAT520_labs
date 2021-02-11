package main

import (
	nl "dat520/lab3/distributed_leader_detection/networklayer"
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	"flag"
	"fmt"
	"math/rand"
	"net/url"
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

	host := *hostname
	var err error
	if host == "" {
		host, err = os.Hostname()
		_, err := url.ParseRequestURI(host)
		if err != nil {
			host = "localhost"
		}
	}
	check(err)

	p := *port
	if p == 0 {
		//random port
		rand.Seed(time.Now().Unix())
		p = rand.Intn(50000) + 10000
	}
	endpoint := host + ":" + fmt.Sprint(p)

	register(endpoint, *connectto)

	nodeIDs := make([]int, *nodes)
	for i := 0; i < *nodes; i++ {
		nodeIDs[i] = i
	}

	nodeIPs := make(map[int]string)

	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	hbSend := make(chan fd.Heartbeat)
	failuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)
	failuredetector.Start() // this starts a new goroutine
	server, err := nl.NewUDPServer(endpoint, failuredetector)
	check(err)
	go server.ServeUDP()
	subchannel := leaderdetector.Subscribe()
	for {
		select {
		case <-subchannel:
			// print leader change?
		case hb := <-hbSend:
			if _, ok := nodeIPs[hb.To]; !ok { // this happens when a request is recieved, which has not been encountered yet, so needs to register
				// add ip to nodeIPs and add id to node IDs?
			}
			// make hb message
			// send on udp
		}
	}
}

func register(ownendpoint, connectto string) {
	nl.SendMessage(connectto, "Register")
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}
