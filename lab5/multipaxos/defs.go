package multipaxos

import (
	bank "dat520/lab5/bank"
	"fmt"
	"net"
)

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
	UniqueID   string
	ClientID   string
	ClientSeq  int
	Noop       bool
	AccountNum int
	Txn        bank.Transaction
	Reconfig   *Reconfig
}

type Reconfig struct {
	ConfigID int
	Ips      []*net.UDPAddr
	Accounts map[int]*bank.Account
	Adu      SlotID
	Include  bool
}

func (r Reconfig) String() string {
	return fmt.Sprintf("ID: %d, Ips: %v, Accounts: %v, Adu: %d, Include: %v", r.ConfigID, r.Ips, r.Accounts, r.Adu, r.Include)
}

func (v Value) Equals(ov Value) bool {
	return v.UniqueID == ov.UniqueID &&
		v.ClientID == ov.ClientID &&
		v.ClientSeq == ov.ClientSeq &&
		v.Noop == ov.Noop &&
		v.AccountNum == ov.AccountNum &&
		v.Txn == ov.Txn &&
		((v.Reconfig == nil && ov.Reconfig == nil )||
		v.Reconfig.ConfigID == ov.Reconfig.ConfigID)
}

// String returns a string representation of value v.
func (v Value) String() string {
	if v.Noop {
		return fmt.Sprintf("No-op value")
	}
	return fmt.Sprintf("Value{ID: %s, ClientID: %s, ClientSeq: %d, Account number: %d, Transaction: %s, Reconfig: %s}",
		v.UniqueID, v.ClientID, v.ClientSeq, v.AccountNum, v.Txn, v.Reconfig)
}

// Response represents a response that can be chosen using the Multi-Paxos algorithm and
// has the following fields:
//
// ClientID: Unique identifier for the client that sent the command.
//
// ClientSeq: Client local sequence number.
//
// TxnRes: The result of the decided transaction.

type Response struct {
	ClientID  string
	ClientSeq int
	TxnRes    bank.TransactionResult
}

// String returns a string representation of response r.
func (r Response) String() string {
	return fmt.Sprintf("Response{ClientID: %s, ClientSeq: %d, Command: %s}",
		r.ClientID, r.ClientSeq, r.TxnRes)
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

func (l *Learn) Equals(ol Learn) bool {
	return l.From == ol.From &&
		l.Slot == ol.Slot &&
		l.Rnd == ol.Rnd &&
		l.Val.Equals(ol.Val)
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
		UniqueID:   "0",
		ClientID:   "1234",
		ClientSeq:  42,
		AccountNum: 1,
	}
	testingValueTwo = Value{
		UniqueID:   "1",
		ClientID:   "5678",
		ClientSeq:  99,
		AccountNum: 2,
	}
	testingValueThree = Value{
		UniqueID:   "2",
		ClientID:   "1369",
		ClientSeq:  4,
		AccountNum: 3,
	}
	testingValueFour = Value{
		UniqueID:   "4",
		ClientID:   "1312",
		ClientSeq:  4,
		AccountNum: 1,
	}
	testingValueFive = Value{
		UniqueID:   "5",
		ClientID:   "4122",
		ClientSeq:  34,
		AccountNum: 2,
	}
	testingValueSix = Value{
		UniqueID:   "6",
		ClientID:   "4512",
		ClientSeq:  245,
		AccountNum: 3,
	}
)
