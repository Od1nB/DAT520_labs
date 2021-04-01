package server_struct

import (
	fd "dat520/lab3/failuredetector"
	ld "dat520/lab3/leaderdetector"
	mp "dat520/lab5/multipaxos"
	nt "dat520/lab5/network"
	"fmt"
	"net"
	"time"
)

var Hello = "Hello"

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
}

func NewServer(id, delay, retryLimit int, addresses []*net.UDPAddr) *Server {
	conn, err := net.ListenUDP("udp", addresses[id])
	nt.Check(err)
	nodeIDs := []int{}
	for i, l := 0, len(addresses); i < l; i++ {
		nodeIDs = append(nodeIDs, i)
	}
	leaderdetector := ld.NewMonLeaderDetector(nodeIDs)
	hbSend := make(chan fd.Heartbeat)
	decidedOut := make(chan mp.DecidedValue, 512)
	promiseOut := make(chan mp.Promise, 512)
	learnOut := make(chan mp.Learn, 512)
	preOut := make(chan mp.Prepare, 512)
	accOut := make(chan mp.Accept, 512)
	numNodes := len(addresses)

	return &Server{
		id:              id,
		conn:            conn,
		servers:         addresses,
		decidedValues:   make(map[string][]mp.DecidedValue),
		clients:         make(map[string]*net.UDPAddr),
		lc:              make(chan nt.Message),
		nodeIDs:         nodeIDs,
		hbSend:          hbSend,
		leaderdetector:  leaderdetector,
		failuredetector: fd.NewEvtFailureDetector(id, nodeIDs, leaderdetector, time.Duration(delay)*time.Millisecond, hbSend),
		proposer:        mp.NewProposer(id, numNodes, -1, leaderdetector, preOut, accOut),
		learner:         mp.NewLearner(id, numNodes, decidedOut),
		acceptor:        mp.NewAcceptor(id, promiseOut, learnOut),
		retryLimit:      retryLimit,
		decidedOut:      decidedOut,
		promiseOut:      promiseOut,
		learnOut:        learnOut,
		preOut:          preOut,
		accOut:          accOut,
	}
}

func (s *Server) StartServerLoop() {
	fmt.Println("Starting server: ", s.servers[s.id], " With id: ", s.id)
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
			s.handleIncomming(&msg)
		case dec := <-s.decidedOut:
			if s.leaderdetector.Leader() != s.id {
				continue
			}
			s.proposer.IncrementAllDecidedUpTo()
			nt.Send(&nt.Message{Tp: 1, DecidedValue: &dec}, s.conn, s.clients[dec.Value.ClientID], s.retryLimit)
		case hb := <-s.hbSend:
			nt.Send(&nt.Message{Tp: 2, Heartbeat: &hb}, s.conn, s.servers[hb.To], s.retryLimit)
		case acc := <-s.accOut:
			nt.Broadcast(&nt.Message{Tp: 3, Accept: &acc}, s.conn, s.servers, s.retryLimit)
		case lrn := <-s.learnOut:
			nt.Broadcast(&nt.Message{Tp: 4, Learn: &lrn}, s.conn, s.servers, s.retryLimit)
		case pre := <-s.preOut:
			nt.Broadcast(&nt.Message{Tp: 5, Prepare: &pre}, s.conn, s.servers, s.retryLimit)
		case prm := <-s.promiseOut:
			nt.Send(&nt.Message{Tp: 6, Promise: &prm}, s.conn, s.servers[prm.To], s.retryLimit)
		}
	}
}

func (s *Server) handleIncomming(msg *nt.Message) {
	switch msg.Tp {
	case 0:
		s.proposer.DeliverClientValue(*msg.Value)
		s.registerClient(msg.Value.ClientID)
	case 2:
		s.failuredetector.DeliverHeartbeat(*msg.Heartbeat)
	case 3:
		s.acceptor.DeliverAccept(*msg.Accept)
	case 4:
		s.learner.DeliverLearn(*msg.Learn)
	case 5:
		s.acceptor.DeliverPrepare(*msg.Prepare)
	case 6:
		s.proposer.DeliverPromise(*msg.Promise)
	}
}

func (s *Server) registerClient(id string) {
	if _, ok := s.clients[id]; !ok {
		addr, err := net.ResolveUDPAddr("udp", id)
		if err != nil {
			fmt.Printf("Malformed id for client: %s", id)
			return
		}
		s.clients[id] = addr
	}
}
