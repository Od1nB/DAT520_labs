package main

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"time"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var myFlags arrayFlags

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
	connecttoid = flag.Int(
		"id",
		0,
		"Id of process to connect to first",
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
	nodeIPs map[int]*net.UDPAddr
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Var(&myFlags, "connect", "Endpoints to connect to")
	flag.Parse()
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	nodeIDs := make([]int, *nodes)
	for i := 0; i < *nodes; i++ {
		nodeIDs[i] = i
	}
	hbSend := make(chan fd.Heartbeat)
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	failuredetector := fd.NewEvtFailureDetector(*id, nodeIDs, leaderdetector, time.Duration(*delay)*time.Millisecond, hbSend)

	//todo: serve.UDP() function
	endpoint, err := net.ResolveUDPAddr("udp", getEndpoint())
	check(err)
	nodeIPs = make(map[int]*net.UDPAddr)
	nodeIPs[*id] = endpoint
	conn, err := net.ListenUDP("udp", endpoint)
	check(err)

	defer conn.Close()

	go listen(conn, failuredetector)
	go subscribePrinter(leaderdetector.Subscribe())

	for {
		hb := <-hbSend
		hbByte := hbToByte(hb)
		conn.WriteToUDP(hbByte, nodeIPs[hb.To])
	}

}

func subscribePrinter(sub <-chan int) {
	for {
		fmt.Println(<-sub)
	}
}

func listen(conn *net.UDPConn, failuredetector *fd.EvtFailureDetector) {
	b := make([]byte, 512, 512)

	for {
		n, _, err := conn.ReadFromUDP(b)
		if err != nil {
			continue
		}

		hb := parseHeartbeat(b[:n])
		// if _, ok := nodeIPs[hb.From]; ok {
		// 	nodeIPs[hb.From] = a

		// }
		failuredetector.DeliverHeartbeat(hb) // todo make real heartbeat
		// 	u.conn.WriteTo(executeCommand(c[0], c[1]), a)
	}
}

func parseHeartbeat(b []byte) fd.Heartbeat {
	return fd.Heartbeat{}
}

func getEndpoint() string {
	host := *hostname
	var err error
	if host == "" {
		host, err = os.Hostname()
		if err != nil {
			host = "localhost"
		} else {
			_, err = url.ParseRequestURI(host)
			if err != nil {
				host = "localhost"
			}
		}
	}

	p := *port
	if p == 0 {
		//random port
		rand.Seed(time.Now().Unix())
		p = rand.Intn(50000) + 10000
	}
	return host + ":" + fmt.Sprint(p)
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}
