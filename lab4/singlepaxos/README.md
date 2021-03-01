# Lab 4: Single-decree Paxos and Multi-Paxos

| Lab 4: | Single-decree Paxos and Multi-Paxos |
| ---------------------    | --------------------- |
| Subject:                 | DAT520 Distributed Systems |
| Deadline:                | **March 15, 2021 23:59** |
| Expected effort:         | 40-60 hours |
| Grading:                 | Pass/fail |
| Submission:              | Group |

## Table of Contents

1. [Algorithm implementation](#algorithm-implementation)

## Algorithm implementation

You will in this task implement the single-decree Paxos algorithm for each of
the three Paxos roles. This task will be verified by Autograder.

The skeleton code, unit tests and definitions for this assignment can be found
in the `singlepaxos` package. Each of the three Paxos roles has a separate
file for skeleton code (e.g. `acceptor.go`) and one for unit tests (e.g.
`acceptor_test.go`). There is additionally a single file called `defs.go`. This
file contains `struct` definitions for the four Paxos messages. You should not
edit this file.

Each of the three Paxos roles have a similar skeleton code structure. They all
have a constructor, e.g. `NewAcceptor(...) *Acceptor`. Each Paxos role also
have a `handle` method for each message type they should receive. For example, the
Acceptor has a method for processing accept messages with the following
signature:

```go
func (a *Acceptor) handleAccept(acc Accept) (lrn Learn, output bool)
```

A `handle` method returns a Paxos message and a boolean value `output`. The
value `output` indicate if handling the message resulted in an output message
from the Paxos module.

For example, if an acceptor handles an accept message and should according to the algorithm reply with a learn message, then the `handleAccept` would return with `output` set to true and the corresponding learn message as `lrn`.
If handling the accept resulted in no outgoing learn message, then `output` should be set to false.
In other words, the caller should _always_ check the value of `output` before eventually using the Paxos message.
If `output` is false, then each field of the Paxos message struct should have the zero value (e.g. initialized using an empty struct literal, `Acceptor{}`).

The `handleLearn` method from `learner.go` does not output a Paxos message.
The return value `val` instead represent a value of which the Paxos nodes had reached
consensus on (i.e. decided).
This value is meant to be used internally on a node to indicate that a value
was chosen. The Value type is defined in `defs.go` as follows:

```go
type Value string

const ZeroValue Value = ""
```

The `Value` definition represents the type of value the Paxos nodes should
agree on.
For simplicity, we define such value as an alias for the `string` type.
It will, in later tasks, be represented something more application specific, e.g. a client request.
A constant named `ZeroValue` is also defined to represent the empty value.

The Paxos message definitions are found in `defs.go`, and shown below, uses the naming conventions found in [this](../resources/paxos-insanely-simple.pdf) algorithm specification (slide 64 and 65).

```go
type Prepare struct {
	From int
	Crnd Round
}

type Promise struct {
	To, From int
	Rnd      Round
	Vrnd     Round
	Vval     Value
}

type Accept struct {
	From int
	Rnd  Round
	Val  Value
}

type Learn struct {
	From int
	Rnd  Round
	Val  Value
}
```

Note that _only_ the `Promise` message struct has a `To` field.
This is because the promise should only be sent to the proposer who sent the corresponding
prepare (unicast).
The other three messages should all be sent to every other Paxos node (broadcast).

The `Round` type definition is also found in `defs.go`, and is type alias for
`int`:

```go
type Round int

const NoRound Round = -1
```

There is also an important constant named `NoRound` defined along with the
type alias.
The constant should be used in promise messages, and more specifically for the `Vrnd` field, to indicate that an acceptor has not voted in any previous round.

In addition to the `handlePromise` method, a Proposer has a method named `increaseCrnd`.
Every proposer maintain, as described in [literature](../README.me#resources), a set of unique round numbers to use when issuing proposals.
A proposer is (for this assignment) defined to have its `crnd` field initially set to the same value as its id.
Note that you need to do a type conversion to do this assignment in the constructor (`Round(id)`).
The `increaseCrnd` method should increase the current `crnd` by the total number of
Paxos nodes.
This is one way to ensure that every proposer uses a disjoint set of round numbers for proposals.

You should for this task implement the following (all marked with `TODO(student)`):

* Any **unexported** field you may need in the `Proposer`, `Acceptor` and
  `Learner` struct.

* The constructor for each of the three Paxos roles: `NewProposer`,
  `NewAcceptor` and `NewLearner`. Note that you do not need to use, or assign
  the channels, for outgoing messages to complete this task, since you are not sending the decided value to anyone, but you can use it to debug your code.

* The `increaseCrnd` method in `proposer.go`.

* The `handlePromise` method in `proposer.go`. _Note:_ The Proposer has a field
  named `clientValue` of type `Value`. The Proposer should in its
  `handlePromise` method use the value of this field as the value to be chosen
  in an eventual outgoing accept message if and only if it has not be bounded
  by another value reported in a quorum of promises.

* The `handlePrepare` and `handleAccept` method in `acceptor.go`.

* The `handleLearn` method in `learner.go`.

Each of the three Paxos roles has a `_test.go` file with unit tests.
_You should not edit these files_.
If you wish to write your own test cases, which is something that we encourage you to do, then do so by creating separate test files.
How to run the complete test suite or an individual test cases has been thoroughly described in previous lab assignments.

The test cases for each Paxos role is a set of actions, more specifically a
sequence of message inputs to the `handle` methods.
The test cases also provide a description of the actual invariant being tested.
You should take a look at the test code to get an understanding of what is going on.
An example of a failing acceptor test case is shown below:

```go
=== RUN TestHandlePrepareAndAccept
--- FAIL: TestHandlePrepareAndAccept (0.00s)
        acceptor_test.go:17: 
                HandlePrepare
                test nr:1
                description: no previous received prepare -> reply with correct rnd and no vrnd/vval
                action nr: 1
                want Promise{To: 1, From: 0, Rnd: 1, No value reported}
                got no output

```

*Note:* This task is solely the core message handling for each Paxos role.
You may (and need) to add fields to each Paxos role struct to maintain Paxos state.
