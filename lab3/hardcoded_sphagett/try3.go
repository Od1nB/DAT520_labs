package main

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
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
	ports = flag.Int(
		"ports",
		20043,
		"Ports for all the servers",
	)
	endpoint = flag.String(
		"endpoint",
		"pitter14.ux.uis.no",
		"Endpoint for this server, only need the IP or machine, port not needed",
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

	hardcoded := [3]string{
		fmt.Sprint("pitter14.ux.uis.no:", *ports),
		fmt.Sprint("pitter16.ux.uis.no:", *ports),
		fmt.Sprint("pitter3.ux.uis.no:", *ports),
	}
	nodeIDs := []int{0, 1, 2}

	addresses := [3]*net.UDPAddr{}
	for i, addr := range hardcoded {
		a, err := net.ResolveUDPAddr("udp", addr)
		check(err)
		addresses[i] = a
	}

	selfAddress, err := net.ResolveUDPAddr("udp", fmt.Sprint(*endpoint, ":", *ports))
	check(err)
	conn, err := net.ListenUDP("udp", selfAddress)
	check(err)

	hbSend := make(chan fd.Heartbeat)
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	failuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)
	failuredetector.Start()

	fmt.Println("Starting server: ", selfAddress, " With id: ", *id)

	defer conn.Close()

	go listen(conn, failuredetector)
	go subscribePrinter(leaderdetector.Subscribe())
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

func subscribePrinter(sub <-chan int) {
	for {
		fmt.Print(<-sub)
		fmt.Println(" Leader change")
	}
}

func listen(conn *net.UDPConn, failuredetector *fd.EvtFailureDetector) {
	b := make([]byte, 512, 512)
	for {
		n, _, err := conn.ReadFromUDP(b)
		if err != nil {
			continue
		}

		hb := fd.Heartbeat{}
		json.Unmarshal(b[:n], &hb)
		// fmt.Println(hb.From, hb.To)
		failuredetector.DeliverHeartbeat(hb) // todo make real heartbeat
		// 	u.conn.WriteTo(executeCommand(c[0], c[1]), a)
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}
