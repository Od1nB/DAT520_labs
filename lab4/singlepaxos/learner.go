package singlepaxos

// Learner represents a learner as defined by the single-decree Paxos
// algorithm.
type Learner struct {
	id        int
	nrOfNodes int
	rnd       Round
	acceptMsg map[Value]int
	froms     []bool
	// Add needed fields
	// Tip: you need to keep the decided values by the Paxos nodes somewhere
}

// NewLearner returns a new single-decree Paxos learner. It takes the
// following arguments:
//
// id: The id of the node running this instance of a Paxos learner.
//
// nrOfNodes: The total number of Paxos nodes.
func NewLearner(id int, nrOfNodes int) *Learner {
	var tem = make([]bool, nrOfNodes)
	return &Learner{
		id:        id,
		nrOfNodes: nrOfNodes,
		acceptMsg: make(map[Value]int),
		rnd:       -1,
		froms:     tem,
	}
}

// Internal: handleLearn processes learn lrn according to the single-decree
// Paxos algorithm. If handling the learn results in learner l emitting a
// corresponding decided value, then output will be true and val contain the
// decided value. If handleLearn returns false as output, then val will have
// its zero value.
func (l *Learner) handleLearn(learn Learn) (val Value, output bool) {
	if l.rnd < learn.Rnd {
		l.rnd = learn.Rnd
		l.froms = make([]bool, l.nrOfNodes)
		l.acceptMsg = make(map[Value]int)
		l.froms[learn.From] = true
		l.acceptMsg[learn.Val]++
	}
	if l.rnd == learn.Rnd && !l.froms[learn.From] {
		l.froms[learn.From] = true
		l.acceptMsg[learn.Val]++
	}
	for v, i := range l.acceptMsg {
		if i >= l.nrOfNodes/2+1 {
			l.froms = make([]bool, l.nrOfNodes)
			l.acceptMsg = make(map[Value]int)
			return v, true
		}
	}
	return "", false
}
