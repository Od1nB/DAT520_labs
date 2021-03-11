package main

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	mp "dat520/lab4/multipaxos"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

var (
	help = flag.Bool(
		"help",
		false,
		"Show usage help",
	)
	client = flag.Bool(
		"client",
		false,
		"Is client instead of server",
	)
	ports = flag.Int(
		"ports",
		20043,
		"Ports for all the servers",
	)
	localhost = flag.Bool(
		"localhost",
		true,
		"Is running on localhost",
	)
	id = flag.Int(
		"id",
		0,
		"Id of this process",
	)
	delay = flag.Int(
		"delay",
		1000,
		"Delay used by Increasing Timout failuredetector in milliseconds",
	)
)

type message struct {
	messageType int
	acc         mp.Accept
	lrn         mp.Learn
	prp         mp.Prepare
	prm         mp.Promise
}

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
	var hardcoded [3]string
	if *localhost {
		hardcoded = [3]string{
			fmt.Sprint("localhost:", *ports),
			fmt.Sprint("localhost:", *ports+1),
			fmt.Sprint("localhost:", *ports+2),
		}
	} else {
		hardcoded = [3]string{
			fmt.Sprint("pitter14.ux.uis.no:", *ports),
			fmt.Sprint("pitter16.ux.uis.no:", *ports),
			fmt.Sprint("pitter3.ux.uis.no:", *ports),
		}
	}
	nodeIDs := []int{0, 1, 2}
	nrNodes := len(nodeIDs)

	addresses := [3]*net.UDPAddr{}
	for i, addr := range hardcoded {
		a, err := net.ResolveUDPAddr("udp", addr)
		check(err)
		addresses[i] = a
	}

	selfAddress, err := net.ResolveUDPAddr("udp", hardcoded[*id])
	check(err)
	conn, err := net.ListenUDP("udp", selfAddress)
	check(err)

	hbSend := make(chan fd.Heartbeat)
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	failuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)
	failuredetector.Start()

	fmt.Println("Starting server: ", selfAddress, " With id: ", *id)

	defer conn.Close()
	preOut := make(chan mp.Prepare)
	accOut := make(chan mp.Accept)
	prp := mp.NewProposer(*id, nrNodes, -1, leaderdetector, preOut, accOut)
	prp.Start()

	decidedOut := make(chan mp.DecidedValue)
	lrn := mp.NewLearner(*id, nrNodes, decidedOut)
	lrn.Start()

	promiseOut := make(chan mp.Promise)
	learnOut := make(chan mp.Learn)
	acc := mp.NewAcceptor(*id, promiseOut, learnOut)
	acc.Start()

	lc := make(chan message)
	go listen(conn, failuredetector, lc)
	for {
		hb := <-hbSend
		// fmt.Println(hb.From, hb.To)
		hbByte, err := json.Marshal(hb)
		if err != nil {
			continue
		}
		conn.WriteToUDP(hbByte, addresses[hb.To])
	}
}

func listen(conn *net.UDPConn, failuredetector *fd.EvtFailureDetector, lc chan message) {
	b := make([]byte, 512, 512)
	for {
		n, _, err := conn.ReadFromUDP(b)
		if err != nil {
			continue
		}
		msg := message{}
		json.Unmarshal(b[:n], &msg)
		lc <- msg
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}
