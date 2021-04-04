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
	c.debug(1, "Starting client: ", c.id)

	defer c.conn.Close()

	go nt.Listen(c.conn, c.lc)

	for {
		text := c.getUserInput()
		if len(text) != 0 {
			c.seq++
			accNum, txn, err := c.getTxn(text)
			if err != "" {
				c.debug(0, err)
				continue
			}
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

		// wait for response
		c.handleResponse()
	}
}

func (c *Client) getUserInput() string {
	fmt.Println("Enter your Account number, Transaction Type and Amount separated by space: ")
	c.scanner.Scan()
	text := c.scanner.Text()
	return text
}

func (c *Client) getTxn(text string) (accNum int, txn *bank.Transaction, e string) {
	splitted := strings.Split(text, " ")

	accNum, err := strconv.Atoi(splitted[0])
	if err != nil {
		return 0, nil, "Account Number can only be numbers!"
	}
	operation := strings.ToUpper(splitted[1])
	switch operation {
	case "BALANCE":
		txn = &bank.Transaction{Op: bank.Balance}
	case "DEPOSIT":
		amount, err := strconv.Atoi(splitted[2])
		if err == nil {
			txn = &bank.Transaction{Op: bank.Deposit, Amount: amount}
		} else {
			e = "Amount can only be numbers!"
		}
	case "WITHDRAW":
		amount, err := strconv.Atoi(splitted[2])
		if err == nil {
			txn = &bank.Transaction{Op: bank.Withdrawal, Amount: amount}
		} else {
			e = "Amount can only be numbers!"
		}
	default:
		e = "Operation can only be: Balance, Deposit, or Withdraw"
	}
	return

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
