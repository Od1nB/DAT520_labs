package multipaxos

import (
	"container/list"
	"dat520/lab3/leaderdetector"
	"fmt"
	"time"
)

// Proposer represents a proposer as defined by the Multi-Paxos algorithm.
type Proposer struct {
	id     int
	quorum int
	n      int

	crnd     Round
	adu      SlotID
	nextSlot SlotID

	promises     []*Promise
	promiseCount int

	phaseOneDone           bool
	phaseOneProgressTicker *time.Ticker

	acceptsOut *list.List
	requestsIn *list.List

	ld     leaderdetector.LeaderDetector
	leader int

	prepareOut chan<- Prepare
	acceptOut  chan<- Accept
	promiseIn  chan Promise
	cvalIn     chan Value

	incDcd chan struct{}
	stop   chan struct{}
}

// NewProposer returns a new Multi-Paxos proposer. It takes the following
// arguments:
//
// id: The id of the node running this instance of a Paxos proposer.
//
// nrOfNodes: The total number of Paxos nodes.
//
// adu: all-decided-up-to. The initial id of the highest _consecutive_ slot
// that has been decided. Should normally be set to -1 initially, but for
// testing purposes it is passed in the constructor.
//
// ld: A leader detector implementing the detector.LeaderDetector interface.
//
// prepareOut: A send only channel used to send prepares to other nodes.
//
// The proposer's internal crnd field should initially be set to the value of
// its id.
func NewProposer(id, nrOfNodes, adu int, ld leaderdetector.LeaderDetector, prepareOut chan<- Prepare, acceptOut chan<- Accept) *Proposer {
	return &Proposer{
		id:     id,
		quorum: (nrOfNodes / 2) + 1,
		n:      nrOfNodes,

		crnd:     Round(id),
		adu:      SlotID(adu),
		nextSlot: 0,

		promises: make([]*Promise, nrOfNodes),

		phaseOneProgressTicker: time.NewTicker(time.Second),

		acceptsOut: list.New(),
		requestsIn: list.New(),

		ld:     ld,
		leader: ld.Leader(),

		prepareOut: prepareOut,
		acceptOut:  acceptOut,
		promiseIn:  make(chan Promise, 8),
		cvalIn:     make(chan Value, 8),

		incDcd: make(chan struct{}),
		stop:   make(chan struct{}),
	}
}

// Start starts p's main run loop as a separate goroutine.
func (p *Proposer) Start() {
	trustMsgs := p.ld.Subscribe()
	go func() {
		for {
			select {
			case prm := <-p.promiseIn:
				accepts, output := p.handlePromise(prm)
				if !output {
					continue
				}
				p.nextSlot = p.adu + 1
				p.acceptsOut.Init()
				p.phaseOneDone = true
				for _, acc := range accepts {
					p.acceptsOut.PushBack(acc)
				}
				p.sendAccept()
			case cval := <-p.cvalIn:
				if p.id != p.leader {
					continue
				}
				p.requestsIn.PushBack(cval)
				if !p.phaseOneDone {
					continue
				}
				p.sendAccept()
			case <-p.incDcd:
				p.adu++
				if p.id != p.leader {
					continue
				}
				if !p.phaseOneDone {
					continue
				}
				p.sendAccept()
			case <-p.phaseOneProgressTicker.C:
				fmt.Println("Leader check", p.id, p.leader, p.phaseOneDone)
				if p.id == p.leader && !p.phaseOneDone {
					p.startPhaseOne()
				}
			case leader := <-trustMsgs:
				p.leader = leader
				if leader == p.id {
					p.startPhaseOne()
				}
			case <-p.stop:
				return
			}
		}
	}()
}

// Stop stops p's main run loop.
func (p *Proposer) Stop() {
	p.stop <- struct{}{}
}

// DeliverPromise delivers promise prm to proposer p.
func (p *Proposer) DeliverPromise(prm Promise) {
	p.promiseIn <- prm
}

// DeliverClientValue delivers client value cval from to proposer p.
func (p *Proposer) DeliverClientValue(cval Value) {
	p.cvalIn <- cval
}

// IncrementAllDecidedUpTo increments the Proposer's adu variable by one.
func (p *Proposer) IncrementAllDecidedUpTo() {
	p.incDcd <- struct{}{}
}

// Internal: handlePromise processes promise prm according to the Multi-Paxos
// algorithm. If handling the promise results in proposer p emitting a
// corresponding accept slice, then output will be true and accs contain the
// accept messages. If handlePromise returns false as output, then accs will be
// a nil slice.
func (p *Proposer) handlePromise(prm Promise) (accs []Accept, output bool) {
	if prm.Rnd == p.crnd && p.isUnique(prm.From) {
		p.promiseCount++
		p.promises = append(p.promises, &prm)
		if p.promiseCount >= p.quorum {
			fmt.Println("Promise count", p.promiseCount, p.quorum)
			return *p.getAccepts(), true
		}
	}
	return nil, false
}

// Internal: increaseCrnd increases proposer p's crnd field by the total number
// of Paxos nodes.
func (p *Proposer) increaseCrnd() {
	p.crnd = p.crnd + Round(p.n)
}

// Internal: startPhaseOne resets all Phase One data, increases the Proposer's
// crnd and sends a new Prepare with Slot as the current adu.
func (p *Proposer) startPhaseOne() {
	fmt.Println("phase 1 try", p.crnd)
	p.phaseOneDone = false
	p.promiseCount = 0
	p.promises = make([]*Promise, p.n)
	p.increaseCrnd()
	p.prepareOut <- Prepare{From: p.id, Slot: p.adu, Crnd: p.crnd}
}

// Internal: sendAccept sends an accept from either the accept out queue
// (generated after Phase One) if not empty, or, it generates an accept using a
// value from the client request queue if not empty. It does not send an accept
// if the previous slot has not been decided yet.
func (p *Proposer) sendAccept() {
	const alpha = 1
	if !(p.nextSlot <= p.adu+alpha) {
		// We must wait for the next slot to be decided before we can
		// send an accept.
		//
		// For Lab 6: Alpha has a value of one here. If you later
		// implement pipelining then alpha should be extracted to a
		// proposer variable (alpha) and this function should have an
		// outer for loop.
		return
	}

	// Pri 1: If bounded by any accepts from Phase One -> send previously
	// generated accept and return.
	if p.acceptsOut.Len() > 0 {
		acc := p.acceptsOut.Front().Value.(Accept)
		p.acceptsOut.Remove(p.acceptsOut.Front())
		p.acceptOut <- acc
		p.nextSlot++
		return
	}

	// Pri 2: If any client request in queue -> generate and send
	// accept.
	if p.requestsIn.Len() > 0 {
		cval := p.requestsIn.Front().Value.(Value)
		p.requestsIn.Remove(p.requestsIn.Front())
		acc := Accept{
			From: p.id,
			Slot: p.nextSlot,
			Rnd:  p.crnd,
			Val:  cval,
		}
		p.nextSlot++
		p.acceptOut <- acc
	}
}

// Internal: isUnique returns true if this is the first promise sent from a node for this round.
func (p *Proposer) isUnique(from int) bool {
	for _, promise := range p.promises {
		if promise != nil && promise.From == from && promise.Rnd == p.crnd {
			return false
		}
	}
	return true
}

// Internal: getAccepts returns a slice of all accept messages for slots between adu and highest slot observed
// If no promise has an accept for a slot, a no-op value is inserted in that slot. The accept messages are in
// order of SlotID
func (p *Proposer) getAccepts() *[]Accept {
	accs := []Accept{}
	promiseSlots := make(map[SlotID]PromiseSlot)
	highestSlotID := SlotID(-1)
	for _, prm := range p.promises {
		if prm != nil {
			for _, slot := range prm.Slots {
				if slot.ID > highestSlotID {
					highestSlotID = slot.ID
				}
				if val, ok := promiseSlots[slot.ID]; !ok || val.Vrnd < slot.Vrnd {
					promiseSlots[slot.ID] = slot
				}
			}
		}
	}

	for i := p.adu + 1; i <= highestSlotID; i++ {
		if slot, ok := promiseSlots[i]; ok {
			accs = append(accs, Accept{p.id, i, p.crnd, slot.Vval})
		} else {
			accs = append(accs, Accept{p.id, i, p.crnd, Value{Noop: true}})
		}
	}
	return &accs
}
