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
		false,
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
	var hardcodedServers []string
	var hardcodedClients []string
	if *localhost {
		hardcodedServers = []string{
			fmt.Sprint("localhost:", *ports),
			fmt.Sprint("localhost:", *ports+1),
			fmt.Sprint("localhost:", *ports+2),
		}
		hardcodedClients = []string{
			fmt.Sprint("localhost:", *ports+3),
			fmt.Sprint("localhost:", *ports+4),
		}
	} else {
		hardcodedServers = []string{
			fmt.Sprint("pitter14.ux.uis.no:", *ports),
			fmt.Sprint("pitter16.ux.uis.no:", *ports),
			fmt.Sprint("pitter3.ux.uis.no:", *ports),
		}
		hardcodedClients = []string{
			fmt.Sprint("pitter1.ux.uis.no:", *ports),
			fmt.Sprint("pitter11.ux.uis.no:", *ports),
		}
		if id == nil {
			host, err := os.Hostname()
			check(err)
			host = fmt.Sprint(host, ":", *ports)
			for i, hn := range hardcodedServers {
				if hn == host {
					*id = i
				}
			}
			for i, hn := range hardcodedClients {
				if hn == host {
					*id = i
				}
			}
			if id == nil {
				os.Exit(1)
			}
		}
	}

	addresses := []*net.UDPAddr{}
	for _, addr := range hardcodedServers {
		a, err := net.ResolveUDPAddr("udp", addr)
		check(err)
		addresses = append(addresses, a)
	}
	clients := []*net.UDPAddr{}
	for _, addr := range hardcodedClients {
		a, err := net.ResolveUDPAddr("udp", addr)
		check(err)
		clients = append(clients, a)
	}
	retryLimit := 3
	lc := make(chan message)

	decidedValues := make(map[string][]mp.DecidedValue)

	if *client {
		rand.Seed(time.Now().Unix())
		myID := *id //rand.Intn((1000 - 2) + 2)
		// myport := 20000 + *id // + myID
		// mySend := make(chan mp.Value)
		// myRec := make(chan mp.Value)
		// var myaddress string
		// if *localhost {
		// 	myaddress = fmt.Sprint("localhost:", myport)
		// }
		// fmt.Println(myt)
		// else{} make myaddress from IP

		// selfAddress, err := net.ResolveUDPAddr("udp", myaddress)
		// check(err)
		cliconn, err := net.ListenUDP("udp", clients[*id])
		check(err)

		fmt.Println("Starting client: ", clients[*id], " With id: ", *id)

		defer cliconn.Close()

		//Put this on own go routine
		//Should pause when no response have been given on a command
		commands := make(map[int]string)
		scanner := bufio.NewScanner(os.Stdin)
		clientSeq := 0
		go listen(cliconn, lc)
		go func() {
			for {
				msg := <-lc
				clientID := msg.DecidedValue.Value.ClientID
				if arr, ok := decidedValues[clientID]; ok {
					if len(arr) < msg.DecidedValue.Value.ClientSeq {
						decidedValues[clientID] = append(arr, *msg.DecidedValue)
					}
				} else {
					decidedValues[clientID] = []mp.DecidedValue{*msg.DecidedValue}
				}
				// fmt.Println("Got message", msg.DecidedValue.Value.Command)
				// decidedValues = append(decidedValues, *msg.DecidedValue)
				fmt.Println("Messages so far:", decidedValues)
			}
		}()
		// go func() {
		for {
			fmt.Println("Enter a command: ")
			scanner.Scan()
			text := scanner.Text()
			if len(text) != 0 {
				clientSeq++
				commands[clientSeq] = text
				m := message{Value: &mp.Value{
					ClientID:  fmt.Sprint(myID),
					ClientSeq: clientSeq,
					Noop:      false,
					Command:   text}}
				broadcast(&m, cliconn, addresses, retryLimit)
			}
		}
		// }()
	} else {

		nodeIDs := []int{0, 1, 2}
		nrNodes := len(nodeIDs)

		// selfAddress, err := net.ResolveUDPAddr("udp", hardcodedServers[*id])
		// check(err)
		conn, err := net.ListenUDP("udp", addresses[*id])
		check(err)

		hbSend := make(chan fd.Heartbeat)
		leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
		failuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)
		failuredetector.Start()

		fmt.Println("Starting server: ", addresses[*id], " With id: ", *id)

		defer conn.Close()
		preOut := make(chan mp.Prepare)
		accOut := make(chan mp.Accept)
		proposer := mp.NewProposer(*id, nrNodes, -1, leaderdetector, preOut, accOut)
		proposer.Start()

		decidedOut := make(chan mp.DecidedValue, 512)
		learner := mp.NewLearner(*id, nrNodes, decidedOut)
		learner.Start()

		promiseOut := make(chan mp.Promise)
		learnOut := make(chan mp.Learn)
		acceptor := mp.NewAcceptor(*id, promiseOut, learnOut)
		acceptor.Start()

		// err = send(&message{Tp: 1, DecidedValue: &mp.DecidedValue{0, mp.Value{"Hello", 0, false, "Hell is closer than you think"}}}, conn, client, 3)
		// check(err)
		go listen(conn, lc)
		// go func() {

		// x := 0
		for {
			// fmt.Println(x)
			// x++
			// fmt.Println("Leader is", leaderdetector.Leader())

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
				fmt.Println(clients)
				broadcast(&message{Tp: 1, DecidedValue: &dec}, conn, clients, retryLimit)
			case hb := <-hbSend:
				// fmt.Println("hb send", hb.To)
				send(&message{Tp: 2, Heartbeat: &hb}, conn, addresses[hb.To], retryLimit)
			case acc := <-accOut:
				fmt.Println("accept out", acc)
				broadcast(&message{Tp: 3, Accept: &acc}, conn, addresses, retryLimit)
			case lrn := <-learnOut:
				fmt.Println("learn out", lrn)
				broadcast(&message{Tp: 4, Learn: &lrn}, conn, addresses, retryLimit)
			case pre := <-preOut:
				fmt.Println("prepare out", pre)
				broadcast(&message{Tp: 5, Prepare: &pre}, conn, addresses, retryLimit)
			case prm := <-promiseOut:
				fmt.Println("promise out", prm)
				send(&message{Tp: 6, Promise: &prm}, conn, addresses[prm.To], retryLimit)
			}
		}
	}
	// }()
	// <-make(chan struct{})
}

func send(msg *message, conn *net.UDPConn, to *net.UDPAddr, retryLimit int) error {
	b, err := json.Marshal(msg)
	// if msg.Tp == 1 {
	// 	fmt.Println(msg.DecidedValue)
	// }
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

func broadcast(msg *message, conn *net.UDPConn, to []*net.UDPAddr, retryLimit int) []error {
	var err []error
	for _, t := range to {
		err = append(err, send(msg, conn, t, retryLimit))
	}
	return err
}

func handleIncomming(msg *message, failuredetector *fd.EvtFailureDetector, leaderdetector *ld.MonLeaderDetector, acceptor *mp.Acceptor, learner *mp.Learner, proposer *mp.Proposer) {
	switch msg.Tp {
	case 0:
		proposer.DeliverClientValue(*msg.Value)
	// case 1:
	// decidedOut <- *msg.DecidedValue
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
	b := make([]byte, 2048, 2048)
	for {
		n, _, err := conn.ReadFromUDP(b)
		// fmt.Println(string(b))
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
