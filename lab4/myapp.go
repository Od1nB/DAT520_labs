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
	val         interface{}
}

func newMessage(messageType int, msg interface{}) message {
	return message{messageType: messageType, val: msg}
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
	proposer := mp.NewProposer(*id, nrNodes, -1, leaderdetector, preOut, accOut)
	proposer.Start()

	decidedOut := make(chan mp.DecidedValue)
	learner := mp.NewLearner(*id, nrNodes, decidedOut)
	learner.Start()

	promiseOut := make(chan mp.Promise)
	learnOut := make(chan mp.Learn)
	acceptor := mp.NewAcceptor(*id, promiseOut, learnOut)
	acceptor.Start()

	retryLimit := 3

	lc := make(chan message)
	go listen(conn, lc)
	for {
		select {
		case msg := <-lc:
			handleIncomming(&msg, failuredetector, acceptor, learner, proposer, decidedOut)
		case dec := <-decidedOut:
			client := &net.UDPAddr{} //todo real client
			send(newMessage(1, dec), conn, client, retryLimit)
		case hb := <-hbSend:
			broadcast(newMessage(2, hb), conn, addresses, retryLimit)
		case acc := <-accOut:
			broadcast(newMessage(3, acc), conn, addresses, retryLimit)
		case lrn := <-learnOut:
			broadcast(newMessage(4, lrn), conn, addresses, retryLimit)
		case pre := <-preOut:
			broadcast(newMessage(5, pre), conn, addresses, retryLimit)
		case prm := <-promiseOut:
			send(newMessage(6, prm), conn, addresses[prm.To], retryLimit)
		}
	}
}

func send(msg message, conn *net.UDPConn, to *net.UDPAddr, retryLimit int) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(b, to)
	for err != nil && retryLimit > 0 {
		_, err = conn.WriteToUDP(b, to)
		retryLimit--
	}
	return err
}

func broadcast(msg message, conn *net.UDPConn, to [3]*net.UDPAddr, retryLimit int) []error {
	var err []error
	for _, t := range to {
		err = append(err, send(msg, conn, t, retryLimit))
	}
	return err
}

func handleIncomming(msg *message, failuredetector *fd.EvtFailureDetector, acceptor *mp.Acceptor, learner *mp.Learner, proposer *mp.Proposer, decidedOut chan mp.DecidedValue) {
	switch msg.messageType {
	case 0:
		v, ok := msg.val.(mp.Value)
		if ok {
			proposer.DeliverClientValue(v)
		}
	case 1:
		v, ok := msg.val.(mp.DecidedValue)
		if ok {
			decidedOut <- v
		}
	case 2:
		v, ok := msg.val.(fd.Heartbeat)
		if ok {
			failuredetector.DeliverHeartbeat(v)
		}
	case 3:
		v, ok := msg.val.(mp.Accept)
		if ok {
			acceptor.DeliverAccept(v)
		}
	case 4:
		v, ok := msg.val.(mp.Learn)
		if ok {
			learner.DeliverLearn(v)
		}
	case 5:
		v, ok := msg.val.(mp.Prepare)
		if ok {
			acceptor.DeliverPrepare(v)
		}
	case 6:
		v, ok := msg.val.(mp.Promise)
		if ok {
			proposer.DeliverPromise(v)
		}
	}
}

func listen(conn *net.UDPConn, lc chan message) {
	b := make([]byte, 512, 512)
	for {
		n, _, err := conn.ReadFromUDP(b)
		if err != nil {
			continue
		}
		msg := message{}
		err = json.Unmarshal(b[:n], &msg)
		if err != nil {
			continue
		}
		lc <- msg
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}
