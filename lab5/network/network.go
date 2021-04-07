package network

import (
	fd "dat520/lab3/failuredetector"
	mp "dat520/lab5/multipaxos"
	"encoding/json"
	"fmt"
	"net"
	"os"
)
type Messagetype int

const (
	Value Messagetype = iota
	DecidedValue
	Heartbeat
	Accept
	Learn
	Prepare
	Promise
	Response
	Reconfig
)

type Message struct {
	Tp Messagetype
	// Val          interface{}
	Value        *mp.Value
	DecidedValue *mp.DecidedValue
	Heartbeat    *fd.Heartbeat
	Accept       *mp.Accept
	Learn        *mp.Learn
	Prepare      *mp.Prepare
	Promise      *mp.Promise
	Response     *mp.Response
	Reconfig 	 *mp.Reconfig
}

func Send(msg *Message, conn *net.UDPConn, to *net.UDPAddr, retryLimit int) error {
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

func Broadcast(msg *Message, conn *net.UDPConn, to []*net.UDPAddr, retryLimit int) []error {
	var err []error
	for _, t := range to {
		err = append(err, Send(msg, conn, t, retryLimit))
	}
	return err
}

func Listen(conn *net.UDPConn, lc chan Message) {
	b := make([]byte, 2048, 2048)
	for {
		n, _, err := conn.ReadFromUDP(b)
		// fmt.Println(string(b))
		if err != nil {
			continue
		}
		msg := Message{}
		err = json.Unmarshal(b[:n], &msg)
		if err != nil {
			continue
		}
		lc <- msg
	}
}

func Check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		os.Exit(1)
	}
}
