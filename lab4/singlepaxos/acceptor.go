package singlepaxos


// Acceptor represents an acceptor as defined by the single-decree Paxos
// algorithm.
type Acceptor struct { 
	id int
	props map[int]*Proposer //Set of proposers
	learns map[int]*Learner //Set of learners
	rnd Round
	vrnd Round
	vval Value

}

// NewAcceptor returns a new single-decree Paxos acceptor.
// It takes the following arguments:
//
// id: The id of the node running this instance of a Paxos acceptor.
func NewAcceptor(id int) *Acceptor {
	return &Acceptor{
		id: id,
		props: make(map[int]*Proposer),
		learns: make(map[int]*Learner),
		rnd: NoRound,
		vrnd: NoRound,
		vval: ZeroValue	,
	}
}

// Internal: handlePrepare processes prepare prp according to the single-decree
// Paxos algorithm. If handling the prepare results in acceptor a emitting a
// corresponding promise, then output will be true and prm contain the promise.
// If handlePrepare returns false as output, then prm will be a zero-valued
// struct.
func (a *Acceptor) handlePrepare(prp Prepare) (prm Promise, output bool) {
	if  prp.Crnd > a.rnd{
		a.rnd = prp.Crnd
		return Promise{prp.From,a.id,a.rnd,a.vrnd,a.vval}, true
	}
	return Promise{},false
}

// Internal: handleAccept processes accept acc according to the single-decree
// Paxos algorithm. If handling the accept results in acceptor a emitting a
// corresponding learn, then output will be true and lrn contain the learn.  If
// handleAccept returns false as output, then lrn will be a zero-valued struct.
func (a *Acceptor) handleAccept(acc Accept) (lrn Learn, output bool) {
	if acc.Rnd >= a.rnd && acc.Rnd != a.vrnd{
		a.rnd = acc.Rnd
		a.vrnd = acc.Rnd
		a.vval = acc.Val
		return Learn{a.id,a.rnd,a.vval},true
	}
	return Learn{}, false
}


