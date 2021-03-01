// Test helper functions - DO NOT EDIT

package singlepaxos

const (
	valueFromClientOne = "A client command"
	valueFromClientTwo = "Another client command"
)

type paction struct {
	promise    Promise
	wantOutput bool
	wantAcc    Accept
}

type msgtype int

const (
	prepare msgtype = iota
	accept
)

type acceptorAction struct {
	msgtype    msgtype
	prepare    Prepare
	accept     Accept
	wantOutput bool
	wantPrm    Promise
	wantLrn    Learn
}

type learnerAction struct {
	learn      Learn
	wantOutput bool
	wantVal    Value
}
