package network

import (
	"errors"
	"fmt"
	"net"
)

var _addressRange = "192.168.1."

var unixAddresses = []string{
	"pitter1.ux.uis.no",
	"pitter3.ux.uis.no",
	"pitter11.ux.uis.no",
	"pitter14.ux.uis.no",
	"pitter16.ux.uis.no",
}

type getAddressFunc func(int, int) string

func GetServerAddresses(version, numberOfNodes, port int) (addresses []*net.UDPAddr, err error) {
	if version < 0 || version > 2 {
		return nil, errors.New("Address version not recognized.")
	}
	var addr *net.UDPAddr
	var f getAddressFunc
	switch version {
	case 0:
		f = getLocalHostAddresses
	case 1:
		f = getUnixAddresses
	case 2:
		f = getDockerAddresses
	}
	for i := 0; i < numberOfNodes; i++ {
		addr, err = net.ResolveUDPAddr("udp", f(port, i))
		if err != nil {
			return
		}
		addresses = append(addresses, addr)
	}
	return
}

func getUnixAddresses(port, i int) string {
	return fmt.Sprintf("%s:%d", unixAddresses[i], port)
}

func getDockerAddresses(port, i int) string {
	return fmt.Sprintf("%s%d:%d", _addressRange, i+1, port)
}

func getLocalHostAddresses(port, i int) string {
	return fmt.Sprintf("localhost:%d", port+i)
}
