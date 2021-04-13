package client_struct

import (
	"bufio"
	"dat520/lab5/bank"
	mp "dat520/lab5/multipaxos"
	nt "dat520/lab5/network"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Client struct {
	id          string
	conn        *net.UDPConn
	scanner     *bufio.Scanner
	seq         int
	servers     []*net.UDPAddr
	lc          chan nt.Message
	retryLimit  int
	debugLevel  int
	selfAddress *net.UDPAddr
	rc          chan *mp.Response
}

func NewClient(id string, retryLimit int, debug int) *Client {
	selfAddress, err := net.ResolveUDPAddr("udp", id)
	nt.Check(err)
	cliconn, err := net.ListenUDP("udp", selfAddress)
	nt.Check(err)
	return &Client{
		id:          id,
		conn:        cliconn,
		scanner:     bufio.NewScanner(os.Stdin),
		seq:         0,
		servers:     []*net.UDPAddr{},
		lc:          make(chan nt.Message),
		retryLimit:  retryLimit,
		debugLevel:  debug,
		selfAddress: selfAddress,
		rc:          make(chan *mp.Response),
	}
}

func (c *Client) getLeader(s string) {
	server, err := net.ResolveUDPAddr("udp", s)
	nt.Check(err)
	nt.Send(&nt.Message{Tp: nt.Servers, Servers: []*net.UDPAddr{c.selfAddress}}, c.conn, server, c.retryLimit)
	msg := <-c.lc
	// ips := strings.Split(msg.Servers," ")
	// for _, ip:= range ips {
	// 	server, err := net.ResolveUDPAddr("udp",ip)
	// 	nt.Check(err)
	// 	c.servers = append(c.servers, server)
	// }
	c.servers = msg.Servers
}

func (c *Client) StartClientLoop(startServer string) {
	c.debug(0, "Starting client: ", c.id)

	defer c.conn.Close()

	go nt.Listen(c.conn, c.lc)
	c.getLeader(startServer)
	go c.handleResponse()
	for {
		text := c.getUserInput()
		if len(text) != 0 {
			c.seq++
			accNum, txn, reconfig, err := c.getTxn(text)
			c.debug(2, "Account nr: ", accNum)
			c.debug(1, txn)
			if err != "" {
				c.debug(0, err)
				continue
			}
			if reconfig != nil {
				v := &mp.Value{
					ClientID:  c.id,
					ClientSeq: c.seq,
					Reconfig:  reconfig,
				}
				v.Reconfig.Include = true
				c.debug(1, v)
				m := nt.Message{Value: v}
				nt.Broadcast(&m, c.conn, c.servers, c.retryLimit)
				v.Reconfig.Include = false
				for _, ip := range v.Reconfig.Ips {
					if !nt.Contains(c.servers, ip) {
						nt.Send(&m, c.conn, ip, c.retryLimit)
					}
				}

			} else {
				v := &mp.Value{
					ClientID:   c.id,
					ClientSeq:  c.seq,
					Noop:       false,
					AccountNum: accNum,
					Txn:        *txn,
				}
				m := nt.Message{Value: v}
				nt.Broadcast(&m, c.conn, c.servers, c.retryLimit)
			}
		}
		// wait for response
		r := <-c.rc
		// for r.TxnRes.ErrorString != "" && len(c.rc) != 0 {
		// 	r = <- c.rc
		// }
		fmt.Println(r.TxnRes)
	}
}

func (c *Client) getUserInput() string {
	fmt.Println("Enter your Transaction Type, Account number, and Amount separated by space: ")
	c.scanner.Scan()
	text := c.scanner.Text()
	return text
}

func (c *Client) getTxn(text string) (accNum int, txn *bank.Transaction, r *mp.Reconfig, e string) {
	splitted := strings.Split(text, " ")

	operation := strings.ToUpper(splitted[0])
	switch operation {
	case "BALANCE":
		txn = &bank.Transaction{Op: bank.Balance}
		var err error
		accNum, err = strconv.Atoi(splitted[1])
		if err != nil {
			return 0, nil, nil, "Account Number can only be numbers!"
		}
	case "DEPOSIT":
		amount, err := strconv.Atoi(splitted[2])
		if err == nil {
			txn = &bank.Transaction{Op: bank.Deposit, Amount: amount}
		} else {
			return 0, nil, nil, "Amount can only be numbers!"
		}
		accNum, err = strconv.Atoi(splitted[1])
		if err != nil {
			return 0, nil, nil, "Account Number can only be numbers!"
		}
	case "WITHDRAW":
		amount, err := strconv.Atoi(splitted[2])
		if err == nil {
			txn = &bank.Transaction{Op: bank.Withdrawal, Amount: amount}
		} else {
			return 0, nil, nil, "Amount can only be numbers!"
		}
		accNum, err = strconv.Atoi(splitted[1])
		if err != nil {
			return 0, nil, nil, "Account Number can only be numbers!"
		}
	case "RECONFIG":
		ipss := splitted[1:]
		ips := make([]*net.UDPAddr, len(ipss))
		for i := range ipss {
			if !strings.Contains(ipss[i], ":") {
				ipss[i] += ":19000"
			}
			addr, err := net.ResolveUDPAddr("udp", ipss[i])
			if err != nil {
				return 0, nil, nil, fmt.Sprintf("Ip did not resolve: %s", ipss[i])
			}
			ips[i] = addr
		}
		r = &mp.Reconfig{Ips: ips}
	case "DEBUG":
		return 0, nil, nil, fmt.Sprintf("Own ip: %v, %s\nServers: %v", c.selfAddress.IP, c.id, c.servers)
	default:
		return 0, nil, nil, "Operation can only be: Balance, Deposit, or Withdraw"
	}
	return accNum, txn, r, e
}

func (c *Client) handleResponse() {
	for {
		msg := <-c.lc
		if msg.Tp == nt.Servers {
			// c.debug(1,"Recieved servers message: ",msg)
			c.servers = msg.Servers
		} else {
			c.rc <- msg.Response
		}
	}

}

func (c *Client) debug(level int, messages ...interface{}) {
	if level <= c.debugLevel {
		fmt.Println(messages...)
	}
}
