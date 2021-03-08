package multipaxos

import "sort"

// Acceptor represents an acceptor as defined by the Multi-Paxos algorithm.
type Acceptor struct {
	// Add needed fields
	id         int
	promiseOut chan<- Promise
	learnOut   chan<- Learn
	stopIn     chan bool
	prepareIn  chan Prepare
	acceptIn   chan Accept
	rnd        Round
	slots      []PromiseSlot
}

// NewAcceptor returns a new Multi-Paxos acceptor.
// It takes the following arguments:
//
// id: The id of the node running this instance of a Paxos acceptor.
//
// promiseOut: A send only channel used to send promises to other nodes.
//
// learnOut: A send only channel used to send learns to other nodes.
func NewAcceptor(id int, promiseOut chan<- Promise, learnOut chan<- Learn) *Acceptor {
	return &Acceptor{
		id:         id,
		promiseOut: promiseOut,
		learnOut:   learnOut,
		stopIn:     make(chan bool),
		prepareIn:  make(chan Prepare),
		acceptIn:   make(chan Accept),
		rnd:        NoRound,
	}
}

// Start starts a's main run loop as a separate goroutine. The main run loop
// handles incoming prepare and accept messages.
func (a *Acceptor) Start() {
	go func() {
		for {
			select {
			case prp := <-a.prepareIn:
				promise, output := a.handlePrepare(prp)
				if output {
					a.promiseOut <- promise
				}
			case acc := <-a.acceptIn:
				learn, output := a.handleAccept(acc)
				if output {
					a.learnOut <- learn
				}
			case <-a.stopIn:
				break
			}
		}
	}()
}

// Stop stops a's main run loop.
func (a *Acceptor) Stop() {
	a.stopIn <- true
}

// DeliverPrepare delivers prepare prp to acceptor a.
func (a *Acceptor) DeliverPrepare(prp Prepare) {
	a.prepareIn <- prp
}

// DeliverAccept delivers accept acc to acceptor a.
func (a *Acceptor) DeliverAccept(acc Accept) {
	a.acceptIn <- acc
}

// Internal: handlePrepare processes prepare prp according to the Multi-Paxos
// algorithm. If handling the prepare results in acceptor a emitting a
// corresponding promise, then output will be true and prm contain the promise.
// If handlePrepare returns false as output, then prm will be a zero-valued
// struct.
func (a *Acceptor) handlePrepare(prp Prepare) (prm Promise, output bool) {
	if prp.Crnd > a.rnd {
		a.rnd = prp.Crnd
		return Promise{prp.From, a.id, a.rnd, a.findSlots(prp.Slot)}, true
	}
	return Promise{}, false
}

// Internal: handleAccept processes accept acc according to the Multi-Paxos
// algorithm. If handling the accept results in acceptor a emitting a
// corresponding learn, then output will be true and lrn contain the learn.  If
// handleAccept returns false as output, then lrn will be a zero-valued struct.
func (a *Acceptor) handleAccept(acc Accept) (lrn Learn, output bool) {
	if acc.Rnd >= a.rnd {
		a.rnd = acc.Rnd
		if !a.replaceAccept(acc) {
			a.slots = append(a.slots, PromiseSlot{acc.Slot, acc.Rnd, acc.Val})
			sort.SliceStable(a.slots, a.sortPromises)
		}
		return Learn{a.id, acc.Slot, acc.Rnd, acc.Val}, true
	}
	return Learn{}, false
}

// Internal: sortPromises sorts promises according to SlotID, ascending
func (a *Acceptor) sortPromises(i, j int) bool {
	return a.slots[i].ID < a.slots[j].ID
}

// Internal: findSlots returns all slots with SlotID higher or equal to argument slotID.
// Return value is a slice or nil, if no larger SlotIDs are found. Equivalent to a filter
// because a.promiseSlots is always sorted.
func (a *Acceptor) findSlots(slotID SlotID) []PromiseSlot {
	for i, l := 0, len(a.slots); i < l; i++ {
		if a.slots[i].ID >= slotID {
			return a.slots[i:]
		}
	}
	return nil
}

// Internal: replaceAccept will replace accept with slotID equal to incomming accept
// if the round of incomming is strictly larger than the current accept.
// Returns true if a value was replaced, else returns false.
func (a *Acceptor) replaceAccept(acc Accept) bool {
	for i, l := 0, len(a.slots); i < l; i++ {
		if a.slots[i].ID > acc.Slot {
			return false // Can stop here because a.slots is always sorted
		}
		if a.slots[i].ID == acc.Slot && a.slots[i].Vrnd < acc.Rnd {
			a.slots[i].Vrnd = acc.Rnd
			a.slots[i].Vval = acc.Val
			return true
		}
	}
	return false
}
