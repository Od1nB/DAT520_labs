package server_struct

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	"dat520/lab5/bank"
	mp "dat520/lab5/multipaxos"
	nt "dat520/lab5/network"
	"fmt"
	"net"
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
	s.debug(1, "Starting server: ", s.servers[s.id], " With id: ", s.id)
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
		s.debug(1, "Incoming Reconfiger: ", msg.Reconfig)
		s.handleReconfigure(*msg.Value)
	}
}

func (s *Server) handleDecidedValue(val *mp.DecidedValue) {
	if val.SlotID > s.adu+1 {
		s.buffer[val.SlotID] = val
		return
	}
	if !val.Value.Noop {
		if val.Value.Reconfig != nil {
			//reconfigFunction()
			s.proposer.Stop()
			s.acceptor.Stop()
			s.learner.Stop()
			s.failuredetector.Stop()

			for i, v := range val.Value.Reconfig.Ips {
				if v == s.servers[s.id] {
					r := val.Value.Reconfig
					nodeIDs := make([]int, len(r.Ips))
					for i := 0; i < len(r.Ips); i++ {
						nodeIDs[i] = i
					}
					s.id = i
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

			return

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
	}
	s.adu++
	s.proposer.IncrementAllDecidedUpTo()
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
	if s.leaderdetector.Leader() == s.id {
		if !v.Reconfig.Include {
			v.Reconfig.Accounts = s.accounts
			v.Reconfig.Adu = s.adu
			v.Reconfig.Include = true
		}
		s.proposer.DeliverClientValue(*v)
		for _, ip := range v.Reconfig.Ips {
			nt.Send(&nt.Message{Tp: nt.Reconfig,Value: v},s.conn,ip,s.retryLimit)
		}
	}
}

func (s *Server) StopServerLoop() {
	s.stop <- struct{}{}
}

func (s *Server) debug(level int, messages ...interface{}) {
	if level <= s.debugLevel {
		fmt.Println(messages...)
	}
}
