package main

import (
	"bufio"
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	mp "dat520/lab4/multipaxos"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
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
		19000,
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
	Tp int
	// Val          interface{}
	Value        *mp.Value
	DecidedValue *mp.DecidedValue
	Heartbeat    *fd.Heartbeat
	Accept       *mp.Accept
	Learn        *mp.Learn
	Prepare      *mp.Prepare
	Promise      *mp.Promise
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

	addresses := [3]*net.UDPAddr{}
	for i, addr := range hardcoded {
		a, err := net.ResolveUDPAddr("udp", addr)
		check(err)
		addresses[i] = a
	}
	retryLimit := 3
	lc := make(chan message)

	if *client {
		rand.Seed(time.Now().Unix())
		myID := rand.Intn((1000 - 2) + 2)
		myport := 20000 // + myID
		// mySend := make(chan mp.Value)
		// myRec := make(chan mp.Value)
		var myaddress string
		if *localhost {
			myaddress := fmt.Sprint("localhost:", myport)
			fmt.Println(myaddress)
		}
		// else{} make myaddress from IP

		selfAddress, err := net.ResolveUDPAddr("udp", myaddress)
		check(err)
		conn, err := net.ListenUDP("udp", selfAddress)
		check(err)

		defer conn.Close()

		//Put this on own go routine
		//Should pause when no response have been given on a command
		commands := make(map[int]string)
		scanner := bufio.NewScanner(os.Stdin)
		clientSeq := 0
		go listen(conn, lc)
		go func() {
			for {
				msg := <-lc
				fmt.Println("Got message", msg.Value.Command)
			}
		}()
		for {
			fmt.Println("Enter a command: ")
			scanner.Scan()
			text := scanner.Text()
			if len(text) != 0 {
				// fmt.Println(text)
				clientSeq++
				commands[clientSeq] = text
				m := message{Value: &mp.Value{
					ClientID:  fmt.Sprint(myID),
					ClientSeq: clientSeq,
					Noop:      false,
					Command:   text}}
				broadcast(&m, conn, addresses, retryLimit)

				//Send to one or all servers
			}
		}
		//Some logic for recieving stuff
	}

	nodeIDs := []int{0, 1, 2}
	nrNodes := len(nodeIDs)

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

	client, _ := net.ResolveUDPAddr("udp", "localhost:20000")
	err = send(&message{Tp: 1, DecidedValue: &mp.DecidedValue{0, mp.Value{"Hello", 0, false, "Hell is closer than you think"}}}, conn, client, 3)
	if err != nil {
		fmt.Println(err)
		os.Exit(12)
	}

	go listen(conn, lc)
	for {
		select {
		case msg := <-lc:
			handleIncomming(&msg, failuredetector, acceptor, learner, proposer, decidedOut)
		case dec := <-decidedOut:
			send(&message{Tp: 1, DecidedValue: &dec}, conn, client, retryLimit)
		case hb := <-hbSend:
			broadcast(&message{Tp: 2, Heartbeat: &hb}, conn, addresses, retryLimit)
		case acc := <-accOut:
			broadcast(&message{Tp: 3, Accept: &acc}, conn, addresses, retryLimit)
		case lrn := <-learnOut:
			broadcast(&message{Tp: 4, Learn: &lrn}, conn, addresses, retryLimit)
		case pre := <-preOut:
			broadcast(&message{Tp: 5, Prepare: &pre}, conn, addresses, retryLimit)
		case prm := <-promiseOut:
			send(&message{Tp: 6, Promise: &prm}, conn, addresses[prm.To], retryLimit)
		}
	}
}

func send(msg *message, conn *net.UDPConn, to *net.UDPAddr, retryLimit int) error {
	b, err := json.Marshal(msg)
	if msg.Tp == 1 {
		fmt.Println(string(b))
	}
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

func broadcast(msg *message, conn *net.UDPConn, to [3]*net.UDPAddr, retryLimit int) []error {
	var err []error
	for _, t := range to {
		err = append(err, send(msg, conn, t, retryLimit))
	}
	return err
}

func handleIncomming(msg *message, failuredetector *fd.EvtFailureDetector, acceptor *mp.Acceptor, learner *mp.Learner, proposer *mp.Proposer, decidedOut chan mp.DecidedValue) {
	switch msg.Tp {
	case 0:
		proposer.DeliverClientValue(*msg.Value)
	case 1:
		decidedOut <- *msg.DecidedValue
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

func listen(conn *net.UDPConn, lc chan message) {
	b := make([]byte, 512, 512)
	for {
		n, _, err := conn.ReadFromUDP(b)
		fmt.Println(n)
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
