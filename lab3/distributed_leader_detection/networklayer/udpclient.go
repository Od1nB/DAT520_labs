package networklayer

import (
	"net"
)

// UDPClient woo
type UDPClient struct {
	conn      *net.UDPConn
	addresses []net.UDPAddr
}

// SendMessage does stuff
func SendMessage(udpAddr, message string) (string, error) {
	addr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return "", err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "", err
	}
	// defer conn.Close()
	var buf [512]byte
	_, err = conn.Write([]byte(message))
	if err != nil {
		return "", err
	}
	n, err := conn.Read(buf[0:])
	if err != nil {
		return "", err
	}
	return string(buf[0:n]), nil
}
