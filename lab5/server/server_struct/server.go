package server_struct

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	"dat520/lab5/bank"
	mp "dat520/lab5/multipaxos"
	nt "dat520/lab5/network"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

type Server struct {
	id              int
	conn            *net.UDPConn
	servers         []*net.UDPAddr
	decidedValues   map[string][]mp.DecidedValue
	clients         map[string]*net.UDPAddr
	lc              chan nt.Message
	nodeIDs         []int
	failuredetector *fd.EvtFailureDetector
	leaderdetector  *ld.MonLeaderDetector
	hbSend          chan fd.Heartbeat
	proposer        *mp.Proposer
	decidedOut      chan mp.DecidedValue
	learner         *mp.Learner
	promiseOut      chan mp.Promise
	learnOut        chan mp.Learn
	acceptor        *mp.Acceptor
	preOut          chan mp.Prepare
	accOut          chan mp.Accept
	retryLimit      int
	debugLevel      int
	accounts        map[int]*bank.Account
	buffer          map[mp.SlotID]*mp.DecidedValue
	adu             mp.SlotID
	sendToClient    chan mp.Response
	stop            chan struct{}
	delay           time.Duration
	reconfigure     bool
}

func NewServer(id, delay, retryLimit int, addresses []*net.UDPAddr, numberOfNodes int, debug int) *Server {
	conn, err := net.ListenUDP("udp", addresses[id])
	nt.Check(err)
	nodeIDs := []int{}
	for i, l := 0, len(addresses); i < l; i++ {
		nodeIDs = append(nodeIDs, i)
	}
	d := time.Duration(delay) * time.Millisecond
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	hbSend := make(chan fd.Heartbeat)
	decidedOut := make(chan mp.DecidedValue, 2048)
	promiseOut := make(chan mp.Promise, 2048)
	learnOut := make(chan mp.Learn, 2048)
	preOut := make(chan mp.Prepare, 2048)
	accOut := make(chan mp.Accept, 2048)
	numNodes := numberOfNodes

	return &Server{
		id:              id,
		conn:            conn,
		servers:         addresses,
		decidedValues:   make(map[string][]mp.DecidedValue),
		clients:         make(map[string]*net.UDPAddr),
		lc:              make(chan nt.Message, 2048),
		nodeIDs:         nodeIDs,
		hbSend:          hbSend,
		leaderdetector:  leaderdetector,
		failuredetector: fd.NewEvtFailureDetector(id, nodeIDs, leaderdetector, d, hbSend),
		proposer:        mp.NewProposer(id, numNodes, -1, leaderdetector, preOut, accOut),
		learner:         mp.NewLearner(id, numNodes, decidedOut),
		acceptor:        mp.NewAcceptor(id, promiseOut, learnOut),
		retryLimit:      retryLimit,
		decidedOut:      decidedOut,
		promiseOut:      promiseOut,
		learnOut:        learnOut,
		preOut:          preOut,
		accOut:          accOut,
		debugLevel:      debug,
		accounts:        make(map[int]*bank.Account),
		buffer:          make(map[mp.SlotID]*mp.DecidedValue),
		sendToClient:    make(chan mp.Response, 2048),
		stop:            make(chan struct{}),
		delay:           d,
	}
}

func (s *Server) StartServerLoop() {
	s.debug(0, "Starting server: ", s.servers[s.id], " With id: ", s.id)
	defer s.conn.Close()
	s.failuredetector.Start()
	s.proposer.Start()
	s.learner.Start()
	s.acceptor.Start()
	go nt.Listen(s.conn, s.lc)
	s.serverLoop()
}

func (s *Server) serverLoop() {
	for {
		select {
		case msg := <-s.lc:
			// debug messages for incomming messages are handled in s.handleIncomming
			s.handleIncomming(&msg)
		case dec := <-s.decidedOut:
			s.handleDecidedValue(&dec)

		case rsp := <-s.sendToClient:
			if s.leaderdetector.Leader() != s.id {
				continue
			}
			s.debug(1, "Sending response value:", rsp, "to", s.clients[rsp.ClientID])
			nt.Send(&nt.Message{Tp: 1, Response: &rsp}, s.conn, s.clients[rsp.ClientID], s.retryLimit)
		case hb := <-s.hbSend:
			s.debug(3, "Sending heartbeat:", hb, "to", s.servers[hb.To])
			nt.Send(&nt.Message{Tp: 2, Heartbeat: &hb}, s.conn, s.servers[hb.To], s.retryLimit)
		case acc := <-s.accOut:
			s.debug(2, "Broadcast accept:", acc)
			nt.Broadcast(&nt.Message{Tp: 3, Accept: &acc}, s.conn, s.servers, s.retryLimit)
		case lrn := <-s.learnOut:
			s.debug(2, "Broadcast learn:", lrn)
			nt.Broadcast(&nt.Message{Tp: 4, Learn: &lrn}, s.conn, s.servers, s.retryLimit)
		case pre := <-s.preOut:
			s.debug(2, "Broadcast prepare:", pre)
			nt.Broadcast(&nt.Message{Tp: 5, Prepare: &pre}, s.conn, s.servers, s.retryLimit)
		case prm := <-s.promiseOut:
			s.debug(2, "Sending promise:", prm, "to", s.servers[prm.To])
			nt.Send(&nt.Message{Tp: 6, Promise: &prm}, s.conn, s.servers[prm.To], s.retryLimit)
		case <-s.stop:
			return
		}
	}
}

func (s *Server) handleIncomming(msg *nt.Message) {
	switch msg.Tp {
	case nt.Value:
		s.debug(1, "Incomming client value:", msg.Value)
		s.proposer.DeliverClientValue(*msg.Value)
		s.registerClient(msg.Value.ClientID)
	case nt.Heartbeat:
		s.debug(3, "Incomming heartbeat:", msg.Heartbeat)
		s.failuredetector.DeliverHeartbeat(*msg.Heartbeat)
	case nt.Accept:
		s.debug(2, "Incomming accept:", msg.Accept)
		s.acceptor.DeliverAccept(*msg.Accept)
	case nt.Learn:
		s.debug(2, "Incomming learn:", msg.Learn)
		s.learner.DeliverLearn(*msg.Learn)
	case nt.Prepare:
		s.debug(2, "Incomming prepare:", msg.Prepare)
		s.acceptor.DeliverPrepare(*msg.Prepare)
	case nt.Promise:
		s.debug(2, "Incomming promise:", msg.Promise)
		s.proposer.DeliverPromise(*msg.Promise)
	case nt.Reconfig:
		s.debug(1, "Incoming reconfiguer: ", msg.Value)
		s.handleReconfigure(msg.Value)
	}
}

func (s *Server) handleDecidedValue(val *mp.DecidedValue) {
	if val.Value.Reconfig.Ips != "" {
		s.proposer.Stop()
		s.debug(1, "Stopped proposer?")
	}
	if val.SlotID > s.adu+1 {
		s.buffer[val.SlotID] = val
		return
	}
	s.debug(1, "Value decided: ", val.Value)
	if !val.Value.Noop {
		if val.Value.Reconfig.Ips != "" {
			s.debug(1, "Processing reconfig: ", val.Value.Reconfig)
			s.proposer.Stop()
			s.acceptor.Stop()
			s.learner.Stop()
			s.failuredetector.Stop()
			delete(s.buffer, s.adu)

			for i, v := range s.buffer {
				s.sendToClient <- mp.Response{
					ClientID:  v.Value.ClientID,
					ClientSeq: v.Value.ClientSeq,
					TxnRes: bank.TransactionResult{
						AccountNum:  v.Value.AccountNum,
						ErrorString: "Reconfiguring please try again.",
					},
				}
				delete(s.buffer, i)
			}
			r := val.Value.Reconfig
			ips := s.splitIps(r.Ips)
			ownIP := s.servers[s.id].String()
			for i, v := range ips {
				s.debug(1, "Not own ip: ", v, s.servers[s.id])
				if v.String() == ownIP {
					s.debug(1, "Found own ip: ", v)
					nodeIDs := make([]int, len(ips))
					for i := 0; i < len(ips); i++ {
						nodeIDs[i] = i
					}
					s.debug(1, nodeIDs)
					s.id = i
					s.adu = r.Adu
					s.servers = ips
					s.leaderdetector = ld.NewMonLeaderDetector(nodeIDs)
					s.failuredetector = fd.NewEvtFailureDetector(s.id, nodeIDs, s.leaderdetector, s.delay, s.hbSend)
					s.proposer = mp.NewProposer(s.id, len(nodeIDs), int(r.Adu), s.leaderdetector, s.preOut, s.accOut)
					s.learner = mp.NewLearner(s.id, len(nodeIDs), s.decidedOut)
					s.acceptor = mp.NewAcceptor(s.id, s.promiseOut, s.learnOut)
					s.failuredetector.Start()
					s.proposer.Start()
					s.acceptor.Start()
					s.learner.Start()
					break
				}
			}
			s.sendToClient <- mp.Response{
				ClientID:  val.Value.ClientID,
				ClientSeq: val.Value.ClientSeq,
				TxnRes: bank.TransactionResult{
					ErrorString: "Reconfig",
				},
			}
			s.reconfigure = false
			s.debug(0, "Reconfiguration done.")
		} else {
			acc, ok := s.accounts[val.Value.AccountNum]
			if !ok {
				acc = &bank.Account{Number: val.Value.AccountNum, Balance: 0}
				s.accounts[val.Value.AccountNum] = acc
			}
			result := acc.Process(val.Value.Txn)
			s.sendToClient <- mp.Response{
				ClientID:  val.Value.ClientID,
				ClientSeq: val.Value.ClientSeq,
				TxnRes:    result,
			}
		}
	} else {
		s.sendToClient <- mp.Response{
			ClientID:  val.Value.ClientID,
			ClientSeq: val.Value.ClientSeq,
			TxnRes: bank.TransactionResult{
				ErrorString: "Noop",
			},
		}
	}
	s.adu++
	s.proposer.IncrementAllDecidedUpTo()
	delete(s.buffer, s.adu)
	if v, ok := s.buffer[s.adu+1]; ok {
		s.handleDecidedValue(v)
	}
}

func (s *Server) registerClient(id string) {
	if _, ok := s.clients[id]; !ok {
		addr, err := net.ResolveUDPAddr("udp", id)
		if err != nil {
			fmt.Printf("Malformed id for client: %s", id)
			return
		}
		s.debug(1, "Registered client:", addr)
		s.clients[id] = addr
	}
}

func (s *Server) handleReconfigure(v *mp.Value) {
	if !s.reconfigure && s.leaderdetector.Leader() == s.id {
		if !v.Reconfig.Include {
			s.debug(1, "Making reconfig")
			accs, err := json.Marshal(s.accounts)
			nt.Check(err)
			v.Reconfig.Accounts = string(accs)
			v.Reconfig.Adu = s.adu
			v.Reconfig.Include = true
			ips := s.splitIps(v.Reconfig.Ips)
			for _, ip := range ips {
				s.debug(1, "Sending reconfig message to: ", ip)
				s.debug(1, "Sending value: ", v)
				nt.Send(&nt.Message{Value: v}, s.conn, ip, s.retryLimit)
			}
		}
		// s.proposer.DeliverClientValue(*v)
	}
}

func (s *Server) splitIps(ips string) (addrs []*net.UDPAddr) {
	ipss := strings.Split(ips, " ")
	for _, ip := range ipss {
		addr, err := net.ResolveUDPAddr("udp", ip)
		nt.Check(err)
		addrs = append(addrs, addr)
	}
	return addrs
}
func (s *Server) marshallAccounts(accs string) (accounts map[int]*bank.Account) {
	err := json.Unmarshal([]byte(accs), &accounts)
	nt.Check(err)
	return accounts
}

func (s *Server) StopServerLoop() {
	s.stop <- struct{}{}
}

func (s *Server) debug(level int, messages ...interface{}) {
	if level <= s.debugLevel {
		fmt.Println(messages...)
	}
}
