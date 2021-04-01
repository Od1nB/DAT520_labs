package main

import (
	"bufio"
	"dat520/lab5/bank"
	mp "dat520/lab5/multipaxos"
	nt "dat520/lab5/network"
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
)

type Client struct {
	clientID string
	conn     *net.UDPConn
	scanner  *bufio.Scanner
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
	rand.Seed(time.Now().Unix())
	myport := 20000 + *id // + myID
	clientID := fmt.Sprintf("%s:%d", *ip, myport)
	selfAddress, err := net.ResolveUDPAddr("udp", clientID)
	nt.Check(err)
	cliconn, err := net.ListenUDP("udp", selfAddress)
	nt.Check(err)
	addresses, err := nt.GetServerAddresses(*version, *numNodes, *ports)
	nt.Check(err)

	fmt.Println("Starting client: ", selfAddress, " With id: ", *id)

	defer cliconn.Close()

	decidedValues := make(map[string][]mp.DecidedValue)
	commands := make(map[int]mp.Value)
	scanner := bufio.NewScanner(os.Stdin)
	clientSeq := 0
	lc := make(chan nt.Message)
	go nt.Listen(cliconn, lc)
	for {
		fmt.Println("Enter a command: ")
		scanner.Scan()
		text := scanner.Text()
		if len(text) != 0 {
			clientSeq++
			v := &mp.Value{
				ClientID:   clientID,
				ClientSeq:  clientSeq,
				Noop:       false,
				AccountNum: 0,
				Txn: bank.Transaction{
					Op:     bank.Deposit,
					Amount: (*id)*100 + clientSeq},
			}
			commands[clientSeq] = *v
			m := nt.Message{Value: v}
			nt.Broadcast(&m, cliconn, addresses, *retryLimit)
		}

		// wait for response
		// TODO server must only send one reply
		msg := <-lc
		clientID := msg.DecidedValue.Value.ClientID
		if arr, ok := decidedValues[clientID]; ok {
			if len(arr) < msg.DecidedValue.Value.ClientSeq {
				decidedValues[clientID] = append(arr, *msg.DecidedValue)
			}
		} else {
			decidedValues[clientID] = []mp.DecidedValue{*msg.DecidedValue}
		}
		fmt.Println("Messages so far:", decidedValues)
	}
}
