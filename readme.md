# HiveMind: A Golang DSM based on TreadMarks

## Overview

HiveMind is an implementation of the TreadMarks DSM as Golang package.  HaveMind implements the core features of TreadMarks, namely full API, Lazy Release Consistency (LRC), multiple-write with lazy diff creation, and garbage collection. HiveMind runs at user level on Linux workstations. No kernel modifications or special privileges are required. As a result, this system is fairly portable.  Because of limited time and resources we will test HiveMind on a cluster of up to 8 nodes on Azure and run a Quicksort application as a benchmark. However, the implementation does not preclude using larger clusters.

## Installing Additional Packages
Hivemind requires the following external packages - [GoVector](https://github.com/DistributedClocks/GoVector), [Shiviz](https://bitbucket.org/bestchai/shiviz) ..

To install GoVector run:
```
go get github.com/DistributedClocks/GoVector
```
Note that GoVector has dependencies on bitbuck repos which uses mercurial. If you get errors the install mercurial using following command:
```
apt-get install mercurial
```

## Running Apps

Make sure all the Azure VMs are online before running HiveMind apps utilizing them.

Jacobi and SimpleDSM are currently using the tipc and can be run locally

Pi is using the tipc but configured for up to 8 Azure VMs. Need to manually launch it at every VM. Specify PID and total drones at launch. IE if launching two:

```
./pi 0 2    <- VM 1
./pi 1 2    <- VM 2
```

Pi_hivemind is configured to run with the proper IPC and utilizes automatic launching of the other drones in the system

Quicksort still has bugs
