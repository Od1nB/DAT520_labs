package leaderdetector

// A MonLeaderDetector represents a Monarchical Eventual Leader Detector as
// described at page 53 in:
// Christian Cachin, Rachid Guerraoui, and Luís Rodrigues: "Introduction to
// Reliable and Secure Distributed Programming" Springer, 2nd edition, 2011.
type MonLeaderDetector struct {
	suspected map[int]bool
	nodeIDs   []int
	leader    int
	channels  []chan int
}

// NewMonLeaderDetector returns a new Monarchical Eventual Leader Detector
// given a list of node ids.
func NewMonLeaderDetector(nodeIDs []int) *MonLeaderDetector {
	m := &MonLeaderDetector{
		suspected: make(map[int]bool),
		nodeIDs:   nodeIDs,
		leader:    UnknownID,
		channels:  make([]chan int, 0),
	}
	return m
}

// Leader returns the current leader. Leader will return UnknownID if all nodes
// are suspected.
func (m *MonLeaderDetector) Leader() int {
	leader := UnknownID
	for _, id := range m.nodeIDs {
		if !m.suspected[id] && id > leader {
			leader = id
		}
	}
	if m.leader != leader {
		for _, channel := range m.channels {
			channel <- leader
		}
	}
	m.leader = leader
	return leader
}

// Suspect instructs the leader detector to consider the node with matching
// id as suspected. If the suspect indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Suspect(id int) {
	m.suspected[id] = true
	m.Leader()
}

// Restore instructs the leader detector to consider the node with matching
// id as restored. If the restore indication result in a leader change
// the leader detector should publish this change to its subscribers.
func (m *MonLeaderDetector) Restore(id int) {
	delete(m.suspected, id)
	m.Leader()
}

// Subscribe returns a buffered channel which will be used by the leader
// detector to publish the id of the highest ranking node.
// The leader detector will publish UnknownID if all nodes become suspected.
// Subscribe will drop publications to slow subscribers.
// Note: Subscribe returns a unique channel to every subscriber;
// it is not meant to be shared.
func (m *MonLeaderDetector) Subscribe() <-chan int {
	channel := make(chan int, 1)
	m.channels = append(m.channels, channel)
	return channel // implicitly converted to read only channel
}
