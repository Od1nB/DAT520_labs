package main

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	mp "dat520/lab5/multipaxos"
	nt "dat520/lab5/network"
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
		19000,
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
	addresses, err := nt.GetServerAddresses(*version, *numNodes, *ports)
	nt.Check(err)

	lc := make(chan nt.Message)

	nodeIDs := []int{}
	for i, l := 0, *numNodes; i < l; i++ {
		nodeIDs = append(nodeIDs, i)
	}

	conn, err := net.ListenUDP("udp", addresses[*id])
	nt.Check(err)

	hbSend := make(chan fd.Heartbeat)
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	failuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)
	failuredetector.Start()

	fmt.Println("Starting server: ", addresses[*id], " With id: ", *id)

	defer conn.Close()
	clients := make(map[string]*net.UDPAddr)
	preOut := make(chan mp.Prepare)
	accOut := make(chan mp.Accept)
	proposer := mp.NewProposer(*id, *numNodes, -1, leaderdetector, preOut, accOut)
	proposer.Start()

	decidedOut := make(chan mp.DecidedValue, 512)
	learner := mp.NewLearner(*id, *numNodes, decidedOut)
	learner.Start()

	promiseOut := make(chan mp.Promise)
	learnOut := make(chan mp.Learn)
	acceptor := mp.NewAcceptor(*id, promiseOut, learnOut)
	acceptor.Start()

	go nt.Listen(conn, lc)

	for {
		select {
		case msg := <-lc:
			// if msg.Tp != 2 {
			// 	fmt.Println("msg incomming", msg)
			// }
			handleIncomming(&msg, failuredetector, leaderdetector, acceptor, learner, proposer)
		case dec := <-decidedOut:
			if leaderdetector.Leader() != *id {
				continue
			}
			fmt.Println("decided out", dec)
			proposer.IncrementAllDecidedUpTo()
			nt.Send(&nt.Message{Tp: 1, DecidedValue: &dec}, conn, clients[dec.Value.ClientID], *retryLimit)
		case hb := <-hbSend:
			nt.Send(&nt.Message{Tp: 2, Heartbeat: &hb}, conn, addresses[hb.To], *retryLimit)
		case acc := <-accOut:
			fmt.Println("accept out", acc)
			nt.Broadcast(&nt.Message{Tp: 3, Accept: &acc}, conn, addresses, *retryLimit)
		case lrn := <-learnOut:
			fmt.Println("learn out", lrn)
			nt.Broadcast(&nt.Message{Tp: 4, Learn: &lrn}, conn, addresses, *retryLimit)
		case pre := <-preOut:
			fmt.Println("prepare out", pre)
			nt.Broadcast(&nt.Message{Tp: 5, Prepare: &pre}, conn, addresses, *retryLimit)
		case prm := <-promiseOut:
			fmt.Println("promise out", prm)
			nt.Send(&nt.Message{Tp: 6, Promise: &prm}, conn, addresses[prm.To], *retryLimit)
		}
	}
}

func handleIncomming(msg *nt.Message, failuredetector *fd.EvtFailureDetector, leaderdetector *ld.MonLeaderDetector, acceptor *mp.Acceptor, learner *mp.Learner, proposer *mp.Proposer) {
	switch msg.Tp {
	case 0:
		proposer.DeliverClientValue(*msg.Value)
	case 2:
		failuredetector.DeliverHeartbeat(*msg.Heartbeat)
	case 3:
		acceptor.DeliverAccept(*msg.Accept)
	case 4:
		learner.DeliverLearn(*msg.Learn)
	case 5:
		acceptor.DeliverPrepare(*msg.Prepare)
	case 6:
		proposer.DeliverPromise(*msg.Promise)
	}
}
