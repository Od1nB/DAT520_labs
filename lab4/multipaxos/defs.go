package multipaxos

import "fmt"

// Type definitions - DO NOT EDIT

// SlotID represents an identifier for a Multi-Paxos consensus instance.
type SlotID int

// Round represents a Multi-Paxos round number.
type Round int

// NoRound is a constant that represents no specific round. It should be used
// as the value for the Vrnd field in Promise messages to indicate that an
// acceptor has not voted in any previous round.
const NoRound Round = -1

// Value represents a value that can be chosen using the Multi-Paxos algorithm.
// type Value string

// ZeroValue is a constant that represents the zero value for the Value type.
// const ZeroValue Value = ""

// Value represents a value that can be chosen using the Multi-Paxos algorithm and
// has the following fields:
//
// ClientID: Unique identifier for the client that sent the command.
//
// ClientSeq: Client local sequence number.
//
// Noop: Boolen to indicate if this Value should be treated as a no-op.
//
// Msg: String message
type Value struct {
	ClientID  string
	ClientSeq int
	Noop      bool
	Command   string
}

// String returns a string representation of value v.
func (v Value) String() string {
	if v.Noop {
		return fmt.Sprintf("No-op value")
	}
	return fmt.Sprintf("Value{ClientID: %s, ClientSeq: %d, Command: %s}",
		v.ClientID, v.ClientSeq, v.Command)
}

// Response represents a response that can be chosen using the Multi-Paxos algorithm and
// has the following fields:
//
// ClientID: Unique identifier for the client that sent the command.
//
// ClientSeq: Client local sequence number.
//
// Command: The decided command.
type Response struct {
	ClientID  string
	ClientSeq int
	Command   string
}

// String returns a string representation of response r.
func (r Response) String() string {
	return fmt.Sprintf("Response{ClientID: %s, ClientSeq: %d, Command: %s}",
		r.ClientID, r.ClientSeq, r.Command)
}

// Message definitions - DO NOT EDIT

// Prepare represents a Multi-Paxos prepare message.
type Prepare struct {
	From int
	Slot SlotID
	Crnd Round
}

// String returns a string representation of prepare p.
func (p Prepare) String() string {
	return fmt.Sprintf("Prepare{From: %d, Slot: %d, Crnd: %d}", p.From, p.Slot, p.Crnd)
}

// Promise represents a Multi-Paxos Paxos promise message.
type Promise struct {
	To, From int
	Rnd      Round
	Slots    []PromiseSlot
}

// String returns a string representation of promise p.
func (p Promise) String() string {
	if p.Slots == nil {
		return fmt.Sprintf("Promise{To: %d, From: %d, Rnd: %d, No values reported (nil slice)}",
			p.To, p.From, p.Rnd)
	}
	if len(p.Slots) == 0 {
		return fmt.Sprintf("Promise{To: %d, From: %d, Rnd: %d, No values reported (empty slice)}",
			p.To, p.From, p.Rnd)
	}
	return fmt.Sprintf("Promise{To: %d, From: %d, Rnd: %d, Slots: %v}",
		p.To, p.From, p.Rnd, p.Slots)
}

// Accept represents a Multi-Paxos Paxos accept message.
type Accept struct {
	From int
	Slot SlotID
	Rnd  Round
	Val  Value
}

// String returns a string representation of accept a.
func (a Accept) String() string {
	return fmt.Sprintf("Accept{From: %d, Slot: %d, Rnd: %d, Val: %v}", a.From, a.Slot, a.Rnd, a.Val)
}

// Learn represents a Multi-Paxos learn message.
type Learn struct {
	From int
	Slot SlotID
	Rnd  Round
	Val  Value
}

// String returns a string representation of learn l.
func (l Learn) String() string {
	return fmt.Sprintf("Learn{From: %d, Slot: %d, Rnd: %d, Val: %v}", l.From, l.Slot, l.Rnd, l.Val)
}

// PromiseSlot represents information about what round and value (if any)
// an acceptor has voted for in slot with id ID
type PromiseSlot struct {
	ID   SlotID
	Vrnd Round
	Vval Value
}

// DecidedValue represents a value decided for a specific slot.
type DecidedValue struct {
	SlotID SlotID
	Value  Value
}

// Testing utilities - DO NOT EDIT

var (
	testingValueOne = Value{
		ClientID:  "1234",
		ClientSeq: 42,
		Command:   "ls",
	}
	testingValueTwo = Value{
		ClientID:  "5678",
		ClientSeq: 99,
		Command:   "rm",
	}
	testingValueThree = Value{
		ClientID:  "1369",
		ClientSeq: 4,
		Command:   "mkdir",
	}
)
