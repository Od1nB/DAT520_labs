package singlepaxos


// Proposer represents a proposer as defined by the single-decree Paxos
// algorithm.
type Proposer struct {
	id int
	crnd        Round
	clientValue Value
	setAcc map[int]*Acceptor
	nrNode int
	highestRound Round
	highestVal Value
	numberAcc int
	quorum int
	seen map[int]bool

}

// NewProposer returns a new single-decree Paxos proposer.
// It takes the following arguments:
//
// id: The id of the node running this instance of a Paxos proposer.
//
// nrOfNodes: The total number of Paxos nodes.
//
// The proposer's internal crnd field should initially be set to the value of
// its id.
func NewProposer(id int, nrOfNodes int) *Proposer {
	return &Proposer{
		id: id,
		crnd: Round(id),
		clientValue: ZeroValue,
		setAcc: make(map[int]*Acceptor,nrOfNodes),
		nrNode: nrOfNodes,
		highestRound: NoRound,
		highestVal: ZeroValue,
		quorum: nrOfNodes/2 + 1,
		seen: make(map[int]bool,nrOfNodes),

	}
}

// Internal: handlePromise processes promise prm according to the single-decree
// Paxos algorithm. If handling the promise results in proposer p emitting a
// corresponding accept, then output will be true and acc contain the promise.
// If handlePromise returns false as output, then acc will be a zero-valued
// struct.
func (p *Proposer) handlePromise(prm Promise) (acc Accept, output bool) {
	if prm.Rnd == p.crnd {
		if ok, _ := p.seen[prm.From]; !ok{
			p.numberAcc++
			p.seen[prm.From] = true
		}
		if prm.Vrnd > p.highestRound{
			p.highestRound = prm.Vrnd
			p.highestVal = prm.Vval
		}
		if p.numberAcc >= p.quorum{
			v := ZeroValue
			if p.highestRound == NoRound {
				v = p.clientValue
			} else{
				v = p.highestVal
			}
			return Accept{p.id,p.crnd,v},true
		}
	}
	return Accept{},false
}

// Internal: increaseCrnd increases proposer p's crnd field by the total number
// of Paxos nodes.
func (p *Proposer) increaseCrnd() {
	p.crnd+= Round(p.nrNode)
}