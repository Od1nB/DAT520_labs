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
	reconfigure     bool
	configID        int
	rc 				chan *mp.Reconfig
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
		proposer:        mp.NewProposer(id, numNodes, -1, 0, leaderdetector, preOut, accOut),
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
		rc: make(chan *mp.Reconfig, 2048),
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
			// s.debug(2,"Incoming configID: ",msg.ConfigID," and own configID: ",s.configID," message type: ",msg.Tp)
			// if msg.ConfigID == s.configID || msg.Tp == nt.Value || msg.Tp == nt.Reconfig{
			if msg.Tp != nt.Heartbeat{
				s.debug(2,"Incoming message ", msg)
			}
			go s.handleIncomming(&msg)
			// }
		case dec := <-s.decidedOut:
			go s.handleDecidedValue(&dec)
		case rsp := <-s.sendToClient:
			if s.leaderdetector.Leader() != s.id {
				continue
			}
			s.debug(1, "Sending response value:", rsp, "to", s.clients[rsp.ClientID])
			nt.Send(&nt.Message{ConfigID:s.configID,Tp: 1, Response: &rsp}, s.conn, s.clients[rsp.ClientID], s.retryLimit)
		case hb := <-s.hbSend:
			s.debug(3, "Sending heartbeat:", hb, "to", s.servers[hb.To])
			nt.Send(&nt.Message{ConfigID:s.configID,Tp: 2, Heartbeat: &hb}, s.conn, s.servers[hb.To], s.retryLimit)
		case acc := <-s.accOut:
			s.debug(2, "Broadcast accept:", acc)
			nt.Broadcast(&nt.Message{ConfigID:s.configID,Tp: 3, Accept: &acc}, s.conn, s.servers, s.retryLimit)
		case lrn := <-s.learnOut:
			s.debug(2, "Broadcast learn:", lrn)
			nt.Broadcast(&nt.Message{ConfigID:s.configID,Tp: 4, Learn: &lrn}, s.conn, s.servers, s.retryLimit)
		case pre := <-s.preOut:
			s.debug(2, "Broadcast prepare:", pre)
			nt.Broadcast(&nt.Message{ConfigID:s.configID,Tp: 5, Prepare: &pre}, s.conn, s.servers, s.retryLimit)
		case prm := <-s.promiseOut:
			s.debug(2, "Sending promise:", prm, "to", s.servers[prm.To])
			nt.Send(&nt.Message{ConfigID:s.configID,Tp: 6, Promise: &prm}, s.conn, s.servers[prm.To], s.retryLimit)
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
		s.debug(1, "Incoming reconfigure: ", msg.Reconfig)
		if s.reconfigure {
			s.rc <- msg.Reconfig
		}
	case nt.Servers:
		s.debug(1, "Incoming servers request: ", msg)
		// sout := make([]string,len(s.servers))
		// for i, servs := range s.servers{
		// 	sout[i] = servs.String()
		// }
		nt.Send(&nt.Message{ConfigID:s.configID,Tp: nt.Servers, Servers: s.servers}, s.conn, msg.Servers[0], s.retryLimit)
	}
}

func (s *Server) handleDecidedValue(val *mp.DecidedValue) {
	if val.SlotID > s.adu+1 {
		s.buffer[val.SlotID] = val
		return
	}
	s.debug(1, "Value decided: ", val.Value)
	if !val.Value.Noop {
		if val.Value.Reconfig != nil {
			s.handleReconfigure(val)
			for i:= range s.buffer {
				delete(s.buffer,i)
			}
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
	s.debug(2, "ADU", s.adu)
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

func (s *Server) handleReconfigure(val *mp.DecidedValue) {
	s.debug(1, "Processing reconfig: ", val.Value.Reconfig)
	s.reconfigure = true
	s.proposer.Stop()
	s.acceptor.Stop()
	s.learner.Stop()
	s.failuredetector.Stop()
	delete(s.buffer, s.adu)
	r := val.Value.Reconfig
	var isIncluded bool
	ownIP := s.servers[s.id].String()
	for i, v := range r.Ips {
		if v.String() == ownIP {
			s.debug(1, "Found own ip: ", v)
			s.initReconfigure(r, i)
			isIncluded = true
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
	if !isIncluded {
		s.StopServerLoop()
		s.debug(0, "Server has stopped.")
	} else {
		servmsg := &nt.Message{ConfigID:s.configID,Tp: nt.Servers, Servers: s.servers}
		for _, cli := range s.clients {
			nt.Send(servmsg, s.conn, cli, s.retryLimit)
		}
	}
}

func (s *Server) initReconfigure(r *mp.Reconfig, id int) {
	nodeIDs := make([]int, len(r.Ips))
	for i := 0; i < len(r.Ips); i++ {
		nodeIDs[i] = i
	}
	s.debug(1, nodeIDs)
	// for i, v := range s.buffer {
	// 	s.sendToClient <- mp.Response{
	// 		ClientID:  v.Value.ClientID,
	// 		ClientSeq: v.Value.ClientSeq,
	// 		TxnRes: bank.TransactionResult{
	// 			AccountNum:  v.Value.AccountNum,
	// 			ErrorString: "Reconfiguring please try again.",
	// 		},
	// 	}
	// 	delete(s.buffer, i)
	// }
	if r.Include {
		s.debug(2,"I am included\nFilling reconfig message")
		r.Accounts = s.accounts
		r.Adu = s.adu
		s.configID++
		r.ConfigID = s.configID
		msg := &nt.Message{ConfigID:s.configID,Tp:nt.Reconfig, Reconfig: r}
		s.debug(2,"Written msg: ",msg)
		for _, ip:= range r.Ips{
			if !nt.Contains(s.servers,ip){
				nt.Send(msg,s.conn,ip,s.retryLimit)
				s.debug(2,"Sending reconfig msg to: ",ip)
			}
		}
	}else{
		s.debug(2,"Waiting for info from other servers")
		rec := <- s.rc
		s.accounts = rec.Accounts
		s.adu = rec.Adu
		s.configID = rec.ConfigID
	}
	s.id = id
	s.servers = r.Ips
	s.leaderdetector = ld.NewMonLeaderDetector(nodeIDs)
	s.failuredetector = fd.NewEvtFailureDetector(s.id, nodeIDs, s.leaderdetector, s.delay, s.hbSend)
	s.proposer = mp.NewProposer(s.id, len(nodeIDs), int(s.adu), s.configID, s.leaderdetector, s.preOut, s.accOut)
	s.learner = mp.NewLearner(s.id, len(nodeIDs), s.decidedOut)
	s.acceptor = mp.NewAcceptor(s.id, s.promiseOut, s.learnOut)
	s.debug(2,"Server has reconfigured all mp modules")
	s.failuredetector.Start()
	s.proposer.Start()
	s.acceptor.Start()
	s.learner.Start()
	s.debug(2,"Initiated all mp modules")
}

// 	if !s.reconfigure && s.leaderdetector.Leader() == s.id {
// 		if !v.Reconfig.Include {
// 			s.debug(1, "Making reconfig")
// 			v.Reconfig.Accounts = s.accounts
// 			v.Reconfig.Adu = s.adu
// 			v.Reconfig.Include = true
// 			v.Reconfig.ConfigID = s.configID + 1
// 			ips := v.Reconfig.Ips
// 			for _, ip := range ips {
// 				s.debug(1, "Sending reconfig message to: ", ip)
// 				s.debug(1, "Sending value: ", v)
// 				nt.Send(&nt.Message{ConfigID:s.configID,Value: v}, s.conn, ip, s.retryLimit)
// 			}
// 		}
// 		// s.proposer.DeliverClientValue(*v)
// 	}
// }

// func (s *Server) splitIps(ips string) (addrs []*net.UDPAddr) {
// 	ipss := strings.Split(ips, " ")
// 	for _, ip := range ipss {
// 		addr, err := net.ResolveUDPAddr("udp", ip)
// 		nt.Check(err)
// 		addrs = append(addrs, addr)
// 	}
// 	return addrs
// }
// func (s *Server) unmarshallAccounts(accs string) (accounts map[int]*bank.Account) {
// 	err := json.Unmarshal([]byte(accs), &accounts)
// 	nt.Check(err)
// 	s.debug(1, "Accounts being marshalled: ", accounts)
// 	return accounts
// }

func (s *Server) StopServerLoop() {
	s.stop <- struct{}{}
}

func (s *Server) debug(level int, messages ...interface{}) {
	if level <= s.debugLevel {
		fmt.Println(messages...)
	}
}
