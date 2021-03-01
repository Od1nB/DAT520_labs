# Lab 4: Single-decree Paxos and Multi-Paxos

| Lab 4: | Single-decree Paxos and Multi-Paxos |
| ---------------------    | --------------------- |
| Subject:                 | DAT520 Distributed Systems |
| Deadline:                | **March 15, 2021 23:59** |
| Expected effort:         | 40-60 hours |
| Grading:                 | Pass/fail |
| Submission:              | Group |

## Table of Contents

- [Lab 4: Single-decree Paxos and Multi-Paxos](#lab-4-single-decree-paxos-and-multi-paxos)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Resources](#resources)
  - [Questions](#questions)
  - [Paxos](#paxos)
  - [Multi-Paxos](#multi-paxos)
  - [Distributed implementation](#distributed-implementation)
  - [Dockerize your application](#dockerize-your-application)
    - [Building the docker image and running the container](#building-the-docker-image-and-running-the-container)

## Introduction

The overall objective of this lab is to implement a single-decree and a multi-decree version of Paxos (also known as Multi-Paxos). The assignment consist of five parts:

1. A set of theory questions that you should answer.

2. Implementation of the single-decree Paxos algorithm for each of the three Paxos roles, (i.e. Proposer, Acceptor and Learner) as described [here](singlepaxos/README.md).
   This implementation has corresponding unit tests and will be verified by Autograder.

3. Implementation of the multi-decree Paxos algorithm for each of the three Paxos roles (i.e. Proposer, Acceptor and Learner) as described [here](multipaxos/README.md).
   This implementation has corresponding unit tests and will be verified by Autograder.

4. Integration of your Multi-Paxos implementation into your application for
   distributed leader detection from [Lab 3](../lab3).
   The goal is to use the failure and leader detection capabilities from your previous implementation to choose a single proposer amongst a group of Paxos processes as the leader for some slot.
   This subtask will be verified by a member of the teaching staff during lab hours.
   
5. The developed application should be containerized with Docker. Your container(s) will be verified by a member of the teaching staff during lab hours.

## Resources

Several Paxos resources are listed below. You should use these resources to
answer the [questions](#questions) for this lab. You are also advised to use
them as support literature when working on your implementation now and in
future lab assignments.

* [Paxos Explained from Scratch](resources/paxos-scratch-slides.pdf)  - slides.
* [Paxos Explained from Scratch ](resources/paxos-scratch-paper.pdf) - paper.
* [Paxos Made Insanely Simple](resources/paxos-insanely-simple.pdf) - slides. Also
  contains pseudo code for Proposer and Acceptor.
* [Paxos Made Simple](resources/paxos-simple.pdf)
* [Paxos Made Moderately Complex](resources/paxos-made-moderately-complex.pdf)
* [Paxos Made Moderately Complex (ACM Computing Surveys)](resources/a42-renesse.pdf)
* [Paxos for System Builders](resources/paxos-system-builders.pdf)
* [The Part-time Parliament](resources/part-time-parliment.pdf)

## Questions

Answer the questions below. You should write down and submit your answers by
using the [answers.md]answers.md) file.

1. Is it possible that Paxos enters an infinite loop? Explain.

2. Is the value to agree on included in the `Prepare` message?

3. Does Paxos rely on an increasing proposal/round number in order to work?
   Explain.

4. Look at this description for Phase 1B: If the proposal number N is larger
   than any previous proposal, then each Acceptor promises not to accept
   proposals less than N, and sends the value it last accepted for this
   instance to the Proposer. What is meant by "the value it last accepted"? And
   what is an "instance" in this case?

5. Explain, with an example, what will happen if there are multiple
   proposers.
   
6. What happens if two proposers both believe themselves to be the leader and
   send `Prepare` messages simultaneously?

7. What can we say about system synchrony if there are multiple proposers (or
   leaders)?

8. Can an acceptor accept one value in round 1 and another value in round 2?
   Explain.

9. What must happen for a value to be "chosen"? What is the connection between
   chosen values and learned values?

## Paxos

The first Paxos implementation for this lab will be a single-decree version as described [here](singlepaxos/README.md).
This variant of Paxos is only expected to choose a single command. It does not need
several slots.

## Multi-Paxos

After implementing the core logic of the single-decree Paxos, your next task is to implement its variation called Multi-Paxos as described [here](multipaxos/README.md), which will allow you to choose multiple commands.

## Distributed implementation

For this task you will integrate your Multi-Paxos implementation together with your network and failure/leader detector from Lab 3.
You will additionally extend your application to handle clients and process multiple commands that are decided by the Multi-Paxos algorithm.
The description of this task can be read [here](multipaxos/README.md#distributed-implementation)

## Dockerize your application

To easy deployment and testing of your application we will use [containers](https://www.docker.com/resources/what-container) to run multiple instances of Paxos nodes. As part of this lab you are expected to complete the installation of Docker and containerize your application.

1. [Docker installation](https://docs.docker.com/install/)
2. [Deploying Go servers with Docker](https://blog.golang.org/docker)

Please, create the dockerfile in the root directory of your application with the following content, assuming that your application is in the directory `lab4/myapp`:

```Docker
# This is a template Dockerfile, please adapt it for your own needs.

# Start from a image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy dependencies from your lab3 files to the container's workspace.
COPY ./lab3 /go/src/app/lab3

# Copy the local package files (lab4) to the container's workspace.
COPY . /go/src/app

WORKDIR /go/src/app

#Build and install your application inside the container.
RUN go install -v ./lab4/myapp

ENTRYPOINT ["/go/bin/myapp"]
```

### Building the docker image and running the container

Go to the root of your group assignment repository (e.g. `dat520-2021/mygroup-labs`) and build your container using the Dockerfile.

```
# Build your container
docker build -t dat520-lab4 -f lab4/Dockerfile .

# Create network
docker network create --subnet=192.168.0.0/16 dat520-net

# Run your container
docker run -itd --name lab4_node1 --net dat520-net --ip 192.168.1.1 --rm dat520-lab4

# Attach standard input, output and error streams
docker attach lab4_node1
```

You can use [docker-compose](https://docs.docker.com/compose/) to build multiple docker images and manage your containers.
