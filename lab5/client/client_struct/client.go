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
	id            string
	conn          *net.UDPConn
	scanner       *bufio.Scanner
	seq           int
	servers       []*net.UDPAddr
	decidedValues map[string][]mp.DecidedValue
	commands      map[int]mp.Value
	lc            chan nt.Message
	retryLimit    int
	debugLevel    int
}

func NewClient(id string, retryLimit int, addresses []*net.UDPAddr, debug int) *Client {
	selfAddress, err := net.ResolveUDPAddr("udp", id)
	nt.Check(err)
	cliconn, err := net.ListenUDP("udp", selfAddress)
	nt.Check(err)
	return &Client{
		id:            id,
		conn:          cliconn,
		scanner:       bufio.NewScanner(os.Stdin),
		seq:           0,
		servers:       addresses,
		decidedValues: make(map[string][]mp.DecidedValue),
		commands:      make(map[int]mp.Value),
		lc:            make(chan nt.Message),
		retryLimit:    retryLimit,
		debugLevel:    debug,
	}
}

func (c *Client) StartClientLoop() {
	c.debug(0, "Starting client: ", c.id)

	defer c.conn.Close()

	go nt.Listen(c.conn, c.lc)

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
					Reconfig:  *reconfig,
				}
				c.debug(1, v)
				c.commands[c.seq] = *v
				m := nt.Message{Tp: nt.Reconfig, Value: v}
				nt.Broadcast(&m, c.conn, c.servers, c.retryLimit)
			} else {
				v := &mp.Value{
					ClientID:   c.id,
					ClientSeq:  c.seq,
					Noop:       false,
					AccountNum: accNum,
					Txn:        *txn,
				}
				c.commands[c.seq] = *v
				m := nt.Message{Value: v}
				nt.Broadcast(&m, c.conn, c.servers, c.retryLimit)
			}
		}

		// wait for response
		c.handleResponse()
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
		r = &mp.Reconfig{Ips: strings.Join(ipss, " ")}
	default:
		return 0, nil, nil, "Operation can only be: Balance, Deposit, or Withdraw"
	}
	return accNum, txn, r, e
}

func (c *Client) handleResponse() {
	msg := <-c.lc
	fmt.Println(msg.Response.TxnRes)
	// clientID := msg.DecidedValue.Value.ClientID
	// if arr, ok := c.decidedValues[clientID]; ok {
	// 	if len(arr) < msg.DecidedValue.Value.ClientSeq {
	// 		c.decidedValues[clientID] = append(arr, *msg.DecidedValue)
	// 	}
	// } else {
	// 	c.decidedValues[clientID] = []mp.DecidedValue{*msg.DecidedValue}
	// }
	// fmt.Println("Messages so far:", c.decidedValues)
}

func (c *Client) debug(level int, messages ...interface{}) {
	if level <= c.debugLevel {
		fmt.Println(messages...)
	}
}
