# Lab 5: Bank with Reconfiguration

| Lab 5: | Bank with Reconfiguration |
| ---------------------    | --------------------- |
| Subject:                 | DAT520 Distributed Systems |
| Deadline:                | **April 9, 2021 23:59** |
| Expected effort:         | 30-40 hours |
| Grading:                 | Pass/fail |
| Submission:              | Group |

## Table of Contents

- [Lab 5: Bank with Reconfiguration](#lab-5-bank-with-reconfiguration)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Task 1: Replicate Bank Accounts](#task-1-replicate-bank-accounts)
    - [Server Module](#server-module)
  - [Task 2: Dynamic Membership through Reconfiguration](#task-2-dynamic-membership-through-reconfiguration)
  - [References](#references)

## Introduction
The overall objective of this lab is to implement a resilient bank application that stores a set of bank accounts and apply transactions to them as they are decided by your Multi-Paxos nodes from the [lab4](https://github.com/dat520-2021/assignments/tree/master/lab4/multipaxos). The assignment consists of two parts:

1. You will use your implementation from previous labs to replicate a set of bank accounts. You will also be required to extend your application's client handling abilities. 

2. You will extend your implementation to enable dynamic membership into your Multi-Paxos protocol, allowing your system to reconfigure the set of nodes available and to keep running in the presence of failures of some nodes.

**Note** that no part of this lab will be verified by Autograder, and they will be verified by a member of the teaching staff during lab hours.

## Task 1: Replicate Bank Accounts
For this task, you should change your implementation for Multi-Paxos in a way to be able to handle bank transactions. The `Value` type definition has changed to the following struct definition:

```go
type Value struct {
    ClientID    string
    ClientSeq   int 
    Noop        bool 
    AccountNum  int
    Txn         bank.Transaction
}
```

The actual value is not anymore a `Command` as in the previous lab, but it is now a combination of the `AccountNum` field and the `Txn` field of type `Transaction` from the `bank` package.
You will also need to extend your main application to store a set of replicated bank accounts and apply transactions to them as they are decided by your Multi-Paxos nodes.

Your system should in this assignment use the Multi-Paxos protocol to replicate a set of bank accounts and their balances information.
Clients can issue transactions to the accounts.
Transactions sent by clients should be sequenced in the same order by all nodes using the Multi-Paxos protocol you have implemented.

Your network layer from the previous assignments should be extended to handle the overall requirements of this lab.
The Failure and Leader detector should **not** need any changes.

### Server Module

You will need to extend your server from lab 4 and add the following functionalities:

* Store bank accounts.
* Apply bank transactions in correct order as they are decided.

All bank related code can be found in the file `bank.go` from the `bank` package. Please take a look at the comments in `bank.go` for the exact semantics. Three structs are defined in the package: `Transaction`, `TransactionResult` and `Account`. A `Transaction` is sent by clients together with an account number in the `Value` struct described earlier. The following method is defined for an Account:

```go
func (a *Account) Process(txn Transaction) TransactionResult
```

The method should be used by a Paxos node when it applies a decided transaction to an account. The resulting `TransactionResult` should be used by a node in the reply to the client. In Lab 4 the reply to a client was simply the value it originally sent. For this lab the reply is the result of applying a client's transaction. The `Response` should therefore been modified in `defs.go` from the `multipaxos` package to use the transaction result instead of a `Command`:

```go
type Response struct {
    ClientID    string
    ClientSeq   int
    TxnRes      bank.TransactionResult
}
```

A node should generate a response after applying a transaction to an account. The `ClientID`, `ClientSeq` and `AccountNum` fields should be populated from the corresponding decided value. A response should be forwarded to the client handling part of your application. It should there be sent to the appropriate client if it is connected.

A learner may, as specified previously, deliver decided values out of order. 
The below pseudo code is similar to the one used in lab 4, but it now performs some checks when applying a transaction.

The server module needs to ensure that only decided values (transactions) are processed in order. The server needs to keep track of the id for the highest decided slot to accomplish this. This assignment, as in lab 4, will use the name `adu` (_all decided up to_) when referring to this variable. It should initially be set to `-1`.

The server should buffer out-of-order decided values and only apply them when it has consecutive sequence of decided slots. More specifically the server module should handle decided values from the learner equivalently to the logic in the following pseudo code:

```go
on receive decided value v from learner:
	handleDecideValue(v)
```

```go
handleDecidedValue(value):
	if slot id for value is larger than adu+1:
		buffer value
		return
	if value is not a no-op:
		if account for account number in value is not found:
			create and store new account with a balance of zero
		apply transaction from value to account if possible (e.g. user has balance)
		create response with appropriate transaction result, client id and client seq
		forward response to client handling module
	increment adu by 1
	increment decided slot for proposer
	if has previously buffered value for (adu+1):
		handleDecidedValue(value from slot adu+1)
```

**Note**: The server should not apply a transaction if the decided value has its `Noop` field set to true. It should for a no-op only increment its `adu` and call the `IncrementAllDecidedUpTo()` method on the Proposer.

You should modify your client command-line interface to allow send a bank transaction to the system by asking the user for the necessary input (account number, transaction type and amount).

## Task 2: Dynamic Membership through Reconfiguration

This task consists of adding dynamic membership into your Multi-Paxos protocol, namely, implement a reconfiguration command to adjust the set of servers that are executing in the system, e.g., adding or replacing nodes while the system is running.
Therefore, a configuration consists of a set of servers that are executing the Paxos protocol.

The Paxos protocol assumes a fixed configuration. Thus we must ensure that a _single configuration_ executes each instance of consensus. 
Reconfiguration can be achieved in different ways.
You could use different configurations for different instances or enabling mechanisms to stop the current execution and perform a migration to the new configuration.

For example, you could implement reconfiguration using different configurations per instance by adding a new special command that defines the set of nodes configured for future instances.
Nodes could agree on this set as part of the consensus, in the same way that they agree for other messages, and once an agreement was reached about the new proposed configuration, all nodes could migrate to such configuration.
Your implementation should also ensure that previous instances that did not get decided were garbage collected, for example, using a `noop` command to force the instances to finalize.

Another alternative would be to prepare your system to migrate its state in case of a reconfiguration, such as creating a snapshot of the system.
In case of adding a new server at slot ```i```, you could stop the consensus for future slots, ensure that you finish executing every lower-numbered slot, obtain the state immediately following execution of slot ```i − 1```, then transfer this latest state to the new server (including application state), and then start the consensus algorithm again in the new configuration.

Please read section 2 of this paper [[1]](#references) for more details of those mechanisms and their advantages and problems.

You should be able to trigger a reconfiguration by sending a special
`reconfiguration request` from the bank client. You will be asked to demonstrate
different types of reconfiguration during the lab approval, e.g., simulating node failures.
Your system should be able to scale from 3 Paxos servers (which handles one failure) to 5 (2 fails) and 7 (3 fails).

* Note that this task is not as simple as it may first seem. We recommend you to study the relevant parts described here [[1]](#references) for guidance and also to take a look at an overview of the technique here [[2]](#references).

## References

1. Leslie Lamport, Dahlia Malkhi, and Lidong Zhou. _Reconfiguring a state
   machine._ SIGACT News, 41(1):63–73, March 2010.
   [[pdf]](resources/reconfig10.pdf)

2. Jacob R. Lorch, Atul Adya, William J. Bolosky, Ronnie Chaiken, John R.
   Douceur, and Jon Howell. _The smart way to migrate replicated stateful
   services._ In Proceedings of the 1st ACM SIGOPS/EuroSys European Conference
   on Computer Systems 2006, EuroSys ’06, pages 103–115, New York, NY, USA, 2006. ACM.
   [[pdf]](resources/smart06.pdf)