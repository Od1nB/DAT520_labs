package multipaxos

// Learner represents a learner as defined by the Multi-Paxos algorithm.
type Learner struct {
	id         int
	nrNodes    int
	slotMap    map[SlotID][]Learn
	rnd        Round
	fromsMap   map[SlotID][]bool
	decidedOut chan<- DecidedValue
	stopIn     chan struct{}
	learnIn    chan Learn
}

// NewLearner returns a new Multi-Paxos learner. It takes the
// following arguments:
//
// id: The id of the node running this instance of a Paxos learner.
//
// nrOfNodes: The total number of Paxos nodes.
//
// decidedOut: A send only channel used to send values that has been learned,
// i.e. decided by the Paxos nodes.
func NewLearner(id int, nrOfNodes int, decidedOut chan<- DecidedValue) *Learner {
	return &Learner{
		id:         id,
		nrNodes:    nrOfNodes,
		slotMap:    make(map[SlotID][]Learn),
		rnd:        NoRound,
		fromsMap:   make(map[SlotID][]bool),
		decidedOut: decidedOut,
		stopIn:     make(chan struct{}),
		learnIn:    make(chan Learn),
	}
}

// Start starts l's main run loop as a separate goroutine. The main run loop
// handles incoming learn messages.
func (l *Learner) Start() {
	go func() {
		for {
			select {
			case lrn := <-l.learnIn:
				val, sid, output := l.handleLearn(lrn)
				if output {
					l.decidedOut <- DecidedValue{SlotID: sid, Value: val}
				}
			case <-l.stopIn:
				break
			}
		}
	}()
}

// Stop stops l's main run loop.
func (l *Learner) Stop() {
	l.stopIn <- struct{}{}
}

// DeliverLearn delivers learn lrn to learner l.
func (l *Learner) DeliverLearn(lrn Learn) {
	l.learnIn <- lrn
}

// Internal: handleLearn processes learn lrn according to the Multi-Paxos
// algorithm. If handling the learn results in learner l emitting a
// corresponding decided value, then output will be true, sid the id for the
// slot that was decided and val contain the decided value. If handleLearn
// returns false as output, then val and sid will have their zero value.
func (l *Learner) handleLearn(learn Learn) (val Value, sid SlotID, output bool) {
	if l.rnd < learn.Rnd {
		l.rnd = learn.Rnd
		l.slotMap = make(map[SlotID][]Learn)
		l.slotMap[learn.Slot] = append(l.slotMap[learn.Slot], learn)
		l.fromsMap = make(map[SlotID][]bool)
		l.fromsMap[learn.Slot] = make([]bool, l.nrNodes)
		l.fromsMap[learn.Slot][learn.From] = true
	}
	if len(l.fromsMap[learn.Slot]) == 0 {
		l.fromsMap[learn.Slot] = make([]bool, l.nrNodes)
	}
	if l.rnd == learn.Rnd && !l.fromsMap[learn.Slot][learn.From] {
		l.slotMap[learn.Slot] = append(l.slotMap[learn.Slot], learn)
		l.fromsMap[learn.Slot][learn.From] = true
	}
	for slt := range l.slotMap {
		vals := make(map[Value]int)
		for lrs := range l.slotMap[slt] {
			vals[l.slotMap[slt][lrs].Val]++
		}
		for v, i := range vals {
			if i >= l.nrNodes/2+1 {
				delete(l.slotMap, slt)
				delete(l.fromsMap, slt)
				return v, slt, true
			}
		}
	}
	return Value{}, 0, false
}
