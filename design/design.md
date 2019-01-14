# HiveMind Design Notes

This design implements the logic outlined in this paper by Keleher et al [TreadMarks: Distributed Shared Memory
on Standard Workstations and Operating Systems](https://www.usenix.org/legacy/publications/library/proceedings/sf94/full_papers/keleher.a), Costa et al - [Lightwight Logging for Lazy Release Consistent Distributed Shared Memory](https://www.usenix.org/legacy/publications/library/proceedings/osdi96/full_papers/costa/index.html) and Zeng -
[Software Distributed Shared Memory System with Fault Detection Support](https://pdfs.semanticscholar.org/1d70/73d2d9564f431e97d424ccde81c40b431ea2.pdf). The pertinent excepts from this paper is copied here.

## Vector Clocks

A [vector clock](https://en.wikipedia.org/wiki/Vector_clock) is an algorithm for generating a partial ordering of events in a distributed system and detecting causality violations. A vector clock of a system of N processes is an array/vector of N logical clocks, one clock per process; a local "smallest possible values" copy of the global clock-array is kept in each process, with the following rules for clock updates:

- Initially all clocks are zero.
- Each time a process experiences an internal event, it increments its own logical clock in the vector by one.
- Each time a process sends a message, it increments its own logical clock in the vector by one (as in the bullet above) and then sends a copy of its own vector.
- Each time a process receives a message, it increments its own logical clock in the vector by one and updates each element in its vector by taking the maximum of the value in its own vector clock and the value in the vector in the received message (for every element).

Definition: "event a" causes "event b" if and only if ta < tb
where:
ta < tb If and only if they meet two conditions:

- they are not equal timestamps ( there exists i, ta[i] != tb[i]) and
- each ta[i] is less than or equal to tb[i] (for all i, ta[i] <= tb[i]) 

### Vector Clock Algebra

1. equal, ta == tb, for all i: ta[i] = tb [i]
2. not equal, ta != tb, for some i: ta[i] != tb[i]
3. less than or equal, ta <= tb: for all i, ta[i] <= tb[i]
4. not less than or equal, ta !<= tb: for some i, ta[i]  !<= tb[i]
5. less than, ta < tb: ta[i] <= tb[i] AND ta[i] != tb[i]
6. not less than, ta !< tb: !(ta[i] <= tb[i] AND ta[i] != tb[i])

### API

*Initial cut at API based on some examples*

- increment(clock, nodeId): increment a vector clock at "nodeId"
- merge(a, b): given two vector clocks, returns a new vector clock with all values greater than those of the merged clocks
- compare(a, b): compare two vector clocks, returns -1 for a < b and 1 for a > b; 0 for concurrent and identical values. 
- isConcurrent(a, b): if A and B are equal, or if they occurred concurrently.
- isIdentical(a, b): if every value in both vector clocks is equal.

Additional [notes](https://pdfs.semanticscholar.org/8dcf/94bcf1bd82344bda990d5f45d7e5978c5e80.pdf) on vector clocks.
More [notes](https://cseweb.ucsd.edu/classes/sp16/cse291-e/applications/ln/lecture4.html)

### Notes from Keleher Thesis
Refer to page 16 of thesis. The following text was taken from thesis and symbols changed to be able to represent them in plain text:

The lazy protocols track which intervals have been performed at a process by maintaining a per process vector timestamp. A vector timestamp consists of a set of interval indices, one per process in the system. Let *vv(i,p)* be the vector timestamp of process *p* at interval *i*. The entry for process *q != p*, denoted *vv(i,p)[q]*, specifies the most recent interval of process *q* that has been performed at process *p*. Entry *vv(i,p)[p]* is equal to *i*.

Interval *I(x,q)*, or interval *x* of processor *q*, is termed **covered** by *vv(i,p)* if *vv(i,p)[q]* is greater than or equal to *x*. We also use the notation *C* to represent covered.

The lazy protocols pass consistency information in the form of write notices that are attached to intervals. A write notice is an indication that a given page has been modified. Each interval contains a write notice for every page that was modified during the segment of time corresponding to the interval. Write notices are used in the send set, which is the set of all write notices created during intervals that have been performed at the releasing process, but not at the acquiring process. If *vv(i,r)* is the vector timestamp of a release process and *vv(j,a)* is the vector timestamp of the corresponding acquire process, then the *send_set* consists of all intervals I(x,p) such that *I(x,p) C vv(i,r)* and not *I(x,p) C vv(j,a)*.In order to create the *send_set*, vector timestamps are included on synchronization requests.




## Runlength Encoding of Diff between Pages

We will use the following algorithm to encode the differences between two pages. Since the pages are the same size, we just need to record the offset in each page that differ and its new value. Therefore, the diff is a list of pairs of (offset,current value).

## 3. LRC and TreadMarks

TreadMarks [Keleher 94] implements a relaxed memory model called lazy release consistency [Keleher 92]. TreadMarks ensures that all programs without data races behave as if they were executing on a conventional sequentially consistent (SC) memory. Most programs satisfy this condition and behave identically in both models, but LRC has the advantage that it can be implemented more efficiently. This section describes TreadMarks' implementation of LRC without the extensions we added to provide fault tolerance.

LRC divides the execution of each process into logical intervals that begin at each synchronization access. Synchronization accesses are classified as release or acquire accesses. Acquiring a lock is an example of an acquire, and releasing a lock is an example of a release. Waiting on a barrier is modeled as a release followed by an acquire. LRC defines the relation corresponds on synchronization accesses as follows: a release access on a lock corresponds to the next acquire on the lock to complete (in real time order); and a release access on a barrier wait corresponds to the acquire accesses executed by all the processes on the same barrier wait.

Intervals are partially ordered according to the transitive closure of the union of the following two relations: (i) intervals on a single process are totally ordered by program order; and (ii) an interval x precedes an interval y, if the release that ends x corresponds to the acquire that starts y. The partial order between intervals is represented by assigning a vector timestamp to each interval. TreadMarks implements lazy release consistency by ensuring that if interval x precedes interval y (according to this partial order), all shared memory updates performed during x are visible at the beginning of y.

### 3.1 Data Structures

Each process in TreadMarks maintains the following data structures in its local volatile memory:

```
pageArray: array with one entry per shared page.
procArray: array with one list of interval records per process.
dirtyList: identifiers of pages that were modified during the current interval.
VC: local vector clock.
pid: local process identifier.

Each SharedPageEntry has fields:
        status: operating system protection status for page (no-access, read-only or read-write)
        twin: original copy of the page
        writeNotices: array with one list of write notice records per process
        manager: identification of the page manager
        copyset: set of processes with copy of the page

Each WriteNoticeRecord has fields:
        diff: pointer to diff
        interval: pointer to corresponding interval record
        pageID: page number

Each IntervalRecord has fields:
        idCreat: id of process which created the interval
        vc: vector time of creator
        writeNotices: list of write notice records for this interval.
```

The status field of a page entry is the operating system protection status for the page, i.e. if the status is no-access then any access to the page triggers a page fault, and if the status is read-only a write access to the page triggers a page fault. The writeNotices field in the page entry describes modifications to the page. The entry for process i in the writeNotices array contains a list with all the write notices created by i for the page, that are known to the local process. Each of these write notice records describes the updates performed by i to the page in a given interval. The write notice record contains a pointer to the interval record describing that interval, and a pointer to a diff containing the words of the page that were updated in the interval. The interval records contain a backpointer to a list with one write notice for each page that was modified during the interval. Whenever an interval record is created, it is tagged with the vector time and the identity of its creator.

The procArray has an entry for each process. The entry for process i contains a list of interval records describing the intervals created by i that the local process knows about. This list is ordered by decreasing interval logical times. We refer to the value of VCi as i's vector time, and to the value of VCi[i] as i's logical time. Similarly, the vector time of an interval created by i is the value of VCi when the interval is created, and the logical time of the interval is the value of `VCi[i]`.

### 3.2 Memory Consistency Algorithm

This subsection describes the implementation of TreadMarks' memory consistency algorithm. The description is based on the pseudo-code for this algorithm presented in Figure 3.1.

LockAcquire is executed when a process tries to acquire a lock and it was not the last process to hold the lock. It sends a request message to the lock manager. The message contains the current vector time of the acquirer. The lock manager forwards the message to the last acquirer.

LockAcquireServer is executed by the releaser when it receives the lock request message. If a new interval has not yet been created since the last local release of this lock, the releaser creates a new interval record for the current interval; and for all the pages modified during the interval it creates write notice records. The diffs encoding the modifications to each of these pages are not created immediately. Instead, they are created lazily when the process receives a diff request or a write notice for a page. The reply message includes a description of all the intervals with timestamps between the acquirer's and the releaser's vector times. We say that an interval i, created by process p, is between the acquirer's and the releaser's vector times if `VCacq[p]<i.vc[p]<=VCrel[p]`.

The description of each interval contains the identifier of the process that created the interval, the vector timestamp of the interval, and the corresponding write notices (remember that interval record = [idCreat, vc, writeNotices]). **Each write notice in the reply contains only the number of a page that was modified during the interval; no updates are transferred in this message.** The information in the reply is obtained by traversing the lists of interval records in procArray. In the reply, each sequence of intervals created by process p is ordered by increasing logical time of p.

The acquirer calls IncorporateIntervals to incorporate the information received in the reply in its own data structures. Notice that the system creates a diff for writable pages for which a write notice was received. This is important to allow the system to distinguish the modifications made by concurrent writers to the page [Carter 91]. The status of the pages for which write notices were received in the reply is set to no-access. When the acquirer tries to access one of the invalidated pages the system invokes the PageFaultHandler. On the first fault for a page, a page copy is requested from the page manager. Pages are kept writable at the manager until the first page request arrives. When this happens, the page status is changed to read-only. After getting the page copy, if there are write notices for the page without the corresponding diffs, the system sends messages requesting those diffs, to the processes that cache them. In TreadMarks, a processor that modified a page in interval i is guaranteed to have all the diffs for that page for all intervals preceding i. After receiving the diffs, the handler applies them to the page in timestamp order. On a read miss, the system provides read-only access to the page. On a write miss, the system provides read-write access and creates a copy of the page (a "twin"), which is used to detect modifications to the page. The twin is later compared to the current contents of the page to produce a diff that encodes the modifications produced during the interval.

The Barrier routine is executed by the system when a process waits on a barrier. If a process is not the manager of the barrier, it sends to the manager its current vector time, plus all intervals between the logical time of the last local interval known at the manager and its current logical time. After this, the manager sends to each other process all the intervals between the current vector time of the process and the manager's current vector time.

The storage for the diffs, the write notice records and the interval records is not freed until a garbage collection is performed, i.e. a process effectively maintains a log of all shared memory accesses since the last garbage collection. During garbage collection, all processes synchronize at a barrier. Then each process updates its local copies of shared pages (either the copy is discarded or all diffs are requested). After this, each process sends a message to the garbage collection manager, informing it that garbage collection is complete. After receiving messages from all other processes, the manager replies to each of them. Each process then frees all data structures used by the LRC protocol and resumes normal execution.

#### LockAcquire:

```
send (AcqRequest, VC) to lock manager;
receive (intervals)
IncorporateIntervals(intervals);
```
#### LockAcquireServer:

```
receive (AcqRequest, VCacq) from acquirer;
wait until lock is released;
if(an interval was not created since the last release of this lock)
        CreateInterval;
send (intervals between VCacq and VC) to acquirer;
```

#### LockRelease:

```
if(lock request pending)
        wakeup LockAcquireServer;
```

#### CreateInterval:

```
if(dirtyList is not empty) {
        VC[pid] := VC[pid] + 1;
        insert new interval i in procArray[pid];
        for each page in dirtyList
                create write notice record for interval i;
        clear dirtyList;
}
```

#### IncorporateIntervals(intervals):

```
for each i in intervals {
        insert record for i in procArray[i.idCreat];
        VC[i.idCreat] := i.vc[i.idCreat];
        for each write notice in i {
          store write notice;
          if (twin exists for the page write notice refers to) {
                if(a write notice correspondingto the current writes was not already created)
                        CreateInterval;
                create diff;
                delete twin;
          }
          set page status to no-access;
        }
}
```

#### DiffRequestServer:

```
receive (DiffRequest, pageId, diffId) from req node;
if (diff is not created) {
        create diff;
        delete twin;
        set page status to read-only;
}
send (diff) to req node;
```


#### PageFaultHandler:

```
if(page status = no-access) {
        if (local page copy not initialized) {
                send (PageRequest, pageId) to page manager;
                receive (page copy, copyset);
        }
        send (DiffRequest, pageId, diffId) to latest writers;
        receive (diffs) from latest writers;
        apply diffs to page in timestamp order;
}
if (write miss) {
        create twin;
        insert page in dirtyList;
        set page status to read-write;
} else
        set page status to read-only;
```

#### PageRequestServer:

```
receive (PageRequest, pageId) from req node;
if (copyset={pid} and page status=read-write)
        set page status to read-only;
copyset := copyset union {req};
send (page copy, copyset) to req node;
```


#### Barrier:

```
CreateInterval;
if (not manager) {
        lastLt := logical time of last local interval known at manager;
        send (Barrier, VC, local intervals between lastLt and VC[pid]) to manager;
        receive (intervals);
        IncorporateIntervals(intervals);
} else {
        for each client c {
                receive (Barrier, VCc, intervals);
                IncorporateIntervals(intervals);
        }
        for each client c
                send(intervals between VCc and VC) to c;
}

```

## Lock Handling

Treadmarks statically assigns the manager for a lock. We will use the following scheme to assign the manager for the locks in the HiveMind implementation. The manager *Ml* for LockID *l* is set as follows:

```
Ml = l mod nrProc
```

Each Node maintains an array of Lock objects indexed by the LockId *l*. The fields of the Lock object are:
| Field | Type | Description | Init Value |
|---|---|---|---|
|local | bool | Set true when this node has the lock Token | true for *Ml* else false |
|held| bool | Set to true if local=true and lock is acquired else false | false |
|lastReq| int | id of the last node to send the LockAcquire to this node | *Ml*  |
|nextID | uint8 | Id of the next node to send the lock to once this node releases the lock. This value is set on first receipt of LockAcquire and this node has lock in locked state| 0 |
|nextVC | *Vclock | Saves the Vclock that was sent by the next LockAcquirer. This value is sent back in the response once the lock is released by this node. | nil |


The LockAcquire method will use the Lock[*l*].lastReq for the destination id for each request. When a node receives the LockAcquire request, there are three cases to handle:
1. node has has lock and lock is unlocked -- send LockAcquire response and release ownership of lock
2. node has lock and lock is locked -- wait until the lock is released before sending back response
3. node does not have lock -- forward the LockAcquire request to Lock[*l*].lastReq

Basically, the LockAcquire request should be passed to the last node in the chain of requests. Each time the lock is released it is passed on to the next node in the chain.

## Barrier Handling

We will set the manager id to 0.

### Multiple Writer Protocol 
From [TreadMarks: Shared Memory Computing on Networks of Workstation](https://pdfs.semanticscholar.org/d9ec/335b3f93b664148b8ce7cff4f1db40ae337b.pdf) section 5.

As the name implies, multiple-writer protocols allow multiple processes to have, at the same time, a writable copy of a page. We explain how a multiple-writer protocol works by referring back to the example of Figure 6 showing a possible memory layout for the Jacobi program (see Figure 3). Assume that both process P1 and P2 initially have an identical valid copy of the page. TreadMarks uses the virtual memory hardware to detect accesses and modifications to shared memory pages (see Figure 8). The shared page is initially write-protected. When a write occurs by P1, TreadMarks creates a copy of the page, or a twin, and saves it as part of the TreadMarks' data structures on P1. It then unprotects the page in the user's address space, so that further writes to the page can occur without software intervention. When P1 arrives at the barrier, we now have the modified copy in the user's address space and the unmodified twin. By doing a word-by-word comparison of the user copy and the twin, we can create a diff, a runlength encoding of the modifications to the page. Once the diff has been created, the twin is discarded. The same sequence of events happens on P2. It is important to note that this entire sequence of events is local to each of the processors, and does not require any message exchanges, unlike in the case of a single-writer protocol. As part
of the barrier mechanism, P1 is informed that P2 has modified the page, and vice versa, and they both invalidate their copy of the page. When they access the page as part of the next iteration, they both take an access fault. The TreadMarks software on P1 knows that P2 has modified the page, sends a message to P2 requesting the diff, and applies that diff to the page when it arrives. Again, the same sequence of events happens on P2, with P1 replaced by P2 and vice versa. With the exception of the first time a processor accesses a page, its copy of the page is updated exclusively by applying diffs: a new complete copy of the page is never needed. The primary benefit of using diffs is that they can be used to implement multiple-writer protocols, thereby reducing the effects of false sharing. In addition, they significantly reduce overall bandwidth requirements because diffs are typically much smaller than a page.

## Inter-processor Messaging 

TreadMarks inter-processor communication used UDP/IP on an Ethernet or an ATM LAN, or through the AAL3/4 protocol on the ATM LAN.  Our implementation will use Ethernet as underlying transport.  We will implement reliable delivery as did Treadmarks with operation-specific, user-level protocols on top of UDP/IP insure delivery.

To minimize latency in handling incoming asynchronous requests, TreadMarks uses a SIGIO signal handler. Message arrival at any socket used to receive request messages generates a SIGIO signal. The handler for UDP/IP avoids the select system call by multiplexing all of the other processors over a single receive socket.  After the handler receives the message, it performs the request and returns. 

### Messages

The following messages are required.

| Message | Msg ID | Payload | Source | Destination |
| --- | --- | --- | --- | --- |
| AcqRequest | LOCKREQ(10) | VC | Lock Acquirer | Lock Holder |
| SendIntervals | LOCKRSP(11) | Intervals between VCacq & VC | Lock Holder | Lock Acquirer |
| DiffRequest | DIFFREQ(20) | pageId, diffId | PageFaultHandler | writers |
| SendDiff | DIFFRSP(21) | diff | writers | PageFaultHandler |
| PageRequest | PAGEREQ(30) | pageId | PageFaultHandler | Page Owner |
| SendPage | PAGERSP(31) | page copy, copyset | Page Owner | PageFaultHandler |
| Barrier | BARRREQ(40) | VC, local Intervals lastLt & VC[pid] | Barrier Client | Barrier Manager |
| SendIntervals | BARRRSP(41) | Intervals between VCc & VC | Barrier Manager |  Barrier Client |

These messages will have the following format:

```
+---------+--------+--------+---------+
| Dest ID | Src ID | Msg ID | Payload |
+---------+--------+--------+---------+
```
Where Dest ID and Src ID is the Pid of the destination and source nodes. The MsgId is Id from the table above and is encoded as byte. The payload is encoded/decoded using the encoding/gob. 

HiveMind will also use an additional Rsp channel that is used to block the Req until the response is received. When a Req is sent out the Req Handler will wait on the channel for a message. When the corresponding Rsp handler receives the response, it will send a message on the Rsp channel. This will unblock the Request and allow it to continue.

#### IPC Design

The IPC layer provides connection handling and message transport for HiveMind DSM. The high-lever architecture is shown ![below](IPCDesign.png). 

The IPC layer is responsible for:
1. Setting up connections to each node
2. Maintaining a connection table for all connections. The Table will maintain a map of IP:Port to NodeID, and status of the connection. 
3. Receive Task for receiving messages from remote nodes. All received messages are forwarded to the Rx Channel. The HiveMind Rx Handler will process messages it receives from the Rx Channel.
4. Send Task to send out messages it receives HiveMind on the Tx Channel. The Send task will lookup the connection from the connection table.
5. Checks the status of the link and updating the status. If there is a failure, it will put a message on the Rx Channel to indicate a failure.



