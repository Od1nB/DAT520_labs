## Answers to Paxos Questions 

You should write down your answers to the
[questions](README.md#questions)
for Lab 4 in this file. 

1. Yes. If two nodes thinks they are the leader and increments rounds
    F.ex. if the network fails between some of the nodes.

2. No.

3. Yes. The proposal is incremented until it "cathches up to" the round number.

4. The Value is the last Value the acceptor has accepted for that round and slot .The instance, where the proposer is the leader.

5. One of them will accept the other as the leader. In an abnormal case both will try to initiate phase until either one realises it should not be the leader, or goes on indefinitely.

6. The one with highest round will get the promise messages from the other nodes. It will then assume leader position for that round.

7. As talked about earlier. The system may halt until a single proposer is elected, or loop indefinitely.

8. Yes. If there is a new slot it will accept the value, if not it will replace the already accepted value for that slot.

9. Chosen value is the value which will be sent back to the client. Learned value is a value given by other acceptors and is used to decide upon the chosen value. When a learner recieves a quorum of learned values it becomes a chosen value.  
