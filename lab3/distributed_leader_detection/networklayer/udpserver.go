package networklayer

// Package networklayer

import (
	fd "dat520/lab3/failuredetector"
	"net"
)

// UDPServer implements the UDP Echo Server specification found at
// https://github.com/COURSE_TAG/assignments/tree/master/lab2/README.md#udp-echo-server
type UDPServer struct {
	conn            *net.UDPConn
	failuredetector *fd.EvtFailureDetector
}

// NewUDPServer returns a new UDPServer listening on addr. It should return an
// error if there was any problem resolving or listening on the provided addr.
func NewUDPServer(addr string, f *fd.EvtFailureDetector) (*UDPServer, error) {
	endpoint, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	connection, err := net.ListenUDP("udp", endpoint)
	if err != nil {
		return nil, err
	}
	return &UDPServer{connection, f}, nil
}

// ServeUDP starts the UDP server's read loop. The server should read from its
// listening socket and handle incoming client requests as according to the
// the specification.
func (u *UDPServer) ServeUDP() {
	defer u.conn.Close()

	b := make([]byte, 512, 512)

	for {
		_, _, err := u.conn.ReadFromUDP(b)
		if err != nil {
			continue
		}
		u.failuredetector.DeliverHeartbeat(fd.Heartbeat{}) // todo make real heartbeat
		// 	u.conn.WriteTo(executeCommand(c[0], c[1]), a)
	}
}
