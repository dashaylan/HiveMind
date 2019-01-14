/*
Package hivemind implements the TreadMarks API.

This file contains the API and the top level handlers for the HiveMind DSM
*/
package hivemind

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"../configs"
	"../ipc"
	"../shm"
	"../tipc"
	"github.com/arcaneiceman/GoVector/govec"
)

// List of Message IDs sent between the HiveMind drones (i.e. nodes)
const (
	LOCKREQ = 10 /* Lock Acquirer 	  -> 	Lock Holder      */
	LOCKRSP = 11 /* Lock Holder 	  -> 	Lock Acquirer    */
	DIFFREQ = 20 /* PagefaultHandler  -> 	Writers 		 */
	DIFFRSP = 21 /* Writers 		  -> 	PageFaultHandler */
	PAGEREQ = 30 /* PageFaultHandler  -> 	Page Owner       */
	PAGERSP = 31 /* Page Owner 		  -> 	PageFaultHandler */
	BARRREQ = 40 /* Barrier Client 	  -> 	Barrier Manager  */
	BARRRSP = 41 /* Barrier Manager   -> 	Barrier Client   */
	/* Message ID 50 Reserved for Connection Message [IPC]   */
)

var msgName = map[uint8]string{
	LOCKREQ: "LOCKREQ", LOCKRSP: "LOCKRSP", DIFFREQ: "DIFFREQ", DIFFRSP: "DIFFRSP",
	PAGEREQ: "PAGEREQ", PAGERSP: "PAGERSP", BARRREQ: "BARRREQ", BARRRSP: "BARRRSP",
}

var LogChan chan string = make(chan string, 100)

// Define the list of structures used for Request/Response handlers
type LockAcquireRequest struct {
	AcqID  uint8 // ID of the Acquirer since this message is forwarded
	LockID uint8
	Vc     Vclock
}

// IntervalRec is a slightly different version of the IntervalRecord
// It is used for the messaging and does not contain list of WriteNotices.
// Instead it contains just the list of modified pages as described in Costa
type IntervalRec struct {
	ProcID uint8
	Vc     Vclock
	Pages  []int
}

type LockAcquireResponse struct {
	LockID    uint8
	Vc        Vclock
	Intervals []IntervalRec
}

type BarrierRequest struct {
	BarrierID uint8
	Vc        Vclock
	Intervals []IntervalRec
}

type BarrierResponse struct {
	Intervals []IntervalRec
	Vc        Vclock
}

type DiffRequest struct {
	Page    int16
	VcStart Vclock
	VcEnd   Vclock
}

type DiffResponse struct {
	Page int16
	WN   []WriteNotice
}

type PageRequest struct {
	Page int16
}

type PageResponse struct {
	Page    int16
	Data    []byte
	CopySet []uint8
}

// HiveMind interface describes the entire HiveMind DSM interface
type HiveMind interface {
	// Allocates a block of free memory and returns the start of that block
	// Can return the following errors:
	// - InsufficientMemory
	Malloc(size int) (int, error)

	// Frees a previously allocated block of memory.
	// Can return the following errors:
	// - InvalidAddress
	// -
	Free(addr int) error

	// Starts the HiveMind DSM on all processors. It should be the first call
	Startup()

	// Exit is called to shutdown the HiveMind DSM.
	Exit()

	// Barrier blocks the calling process until all other processes arrive at the barrier
	Barrier(b uint8)

	// LockAcquire blocks the calling process until it acquires the specified lock
	LockAcquire(lock uint8)

	// LockRelease releases the specified lock
	LockRelease(lock uint8)

	// GetNProcs gets the number of processors configured for this DSM
	GetNProcs() uint8

	// GetProcID gets the processor ID of this process
	GetProcID() uint8

	// Read a single byte at the specified address or Read array of bytes
	// Can return the following errors:
	// -
	Read(addr int) (byte, error)
	ReadN(addr int, len int) ([]byte, error)
	ReadFloat(addr int) (float64, error)

	// Write a byte at address or write an array of bytes starting at address
	// Can return the following errors:
	// -
	Write(addr int, val byte) error
	WriteN(addr int, data []byte) error
	WriteFloat(addr, val float64) error
}

// Lock holds the state of each lock in the system
type Lock struct {
	mutex   *sync.Mutex // Mutex to protect RW to this lock
	local   bool        // Set true when this processor has the lock token
	held    bool        // Tracks the lock state. Set to true when the lock is acquired.
	lastReq uint8       // Last Lock Acquirer
	nextID  uint8       // Next node wanting to get the lock
	nextVC  *Vclock     // Next node timestamp
}

// HM encapsulates the entire HiveMind DSM data
type HM struct {
	pid        uint8             // This processors id
	VC         Vclock            // local vector clock
	shm        *shm.Shm          // Shared Memory block
	nrProc     uint8             // Number of processors in the DSM
	nrLocks    uint8             // Number of locks supported
	nrBarriers uint8             // Number of barriers supported
	procArray  *ProcArray        // array with one list of interval records per process.
	pageArray  *PageArray        // array with one entry per shared page.
	dirtyList  map[int]bool      // identifiers of pages that were modified during the current interval
	dlMutex    *sync.RWMutex     // Mutex to protect read/writes to dirtyList map
	locks      []*Lock           // holds the locks
	txChan     chan<- []byte     // Channel to send messages to IPC
	rxChan     <-chan []byte     // Channel to receive messages from IPC
	ipc        ipc.Ipc           // Connection handle to IPC layer
	waitChan   chan bool         // channel use to block main thread until corresponding event is received
	halt       chan bool         // channel used to signal HM is shutting down
	barrierReq []*BarrierRequest // Array to hold barrier requests from clients
	barrierCnt uint8             // keeps track of number of barrier requests received
	tipc       *tipc.IpcConn     // Temp IPC connection for unit testing
	debugLevel int               // Debug level 0= None, 1=Error, 2=Info 3=Msg 4=Debug
	start      time.Time         // Time when the library is started
	vecLog     *govec.GoLog      // GoVector object
}

//NewHiveMind creates a new instance of the HM struct and initializes data structures
func NewHiveMind(pid, nrProc, nrBarriers, nrLocks uint8, nrPages, pageSize int, ipcType string) *HM {
	// If procAddr is not specified then we plan to use tipc for testing locally
	hm := new(HM)
	if ipcType == "ipc" || ipcType == "" {
		myip := ipc.GetOutboundIP()
		locConf, err := configs.ReadConfig()
		if err != nil {
			//can't read from config
			return nil
		}
		nrProc = uint8(len(locConf.Drones)+1)
		fmt.Println(locConf, hm.nrProc)
		if locConf.DroneList != nil {
			// I am not the first drone
			nrProc = uint8(len(locConf.DroneList)+1)
			for _, drone := range locConf.DroneList {
				if myip == drone.Address {
					//It's me
					pid = drone.PID
					break
				}
			}
		} else {
			// null DroneList means I am the first drone, drone 0
			pid = uint8(0)
		}


	}



	hm.nrLocks = nrLocks
	hm.nrProc = nrProc
	hm.nrBarriers = nrBarriers
	hm.VC = *NewVectorClock(nrProc)
	hm.pid = pid
	hm.dirtyList = make(map[int]bool, 0)
	hm.dlMutex = new(sync.RWMutex)
	hm.shm = shm.NewShm(pageSize, nrPages)
	hm.shm.InstallSEGVHandler(hm.pageFaultHandler)
	hm.procArray = NewProcArray(nrProc)
	hm.pageArray = NewPageArray(nrPages, pageSize, pid, nrProc)
	hm.waitChan = make(chan bool, 1)
	hm.barrierReq = make([]*BarrierRequest, nrProc)
	hm.locks = make([]*Lock, nrLocks)
	hm.start = time.Now()
	return hm
}

// Malloc a block of free memory and returns the start of that block
func (hm *HM) Malloc(size int) (int, error) {
	return hm.shm.Malloc(size)
}

// Free a previously allocated block of memory.
func (hm *HM) Free(addr int) error {
	return hm.shm.Free(addr)
}

// SetDebug sets the debug message level. Lower levels are included in
// higher levels
// 0 - disable all output
// 1 - Enable Error messages
// 2 - Enable Info messages
// 3 - Enables IPC message trace
// 4 - Enable Debug messages
func (hm *HM) SetDebug(level int) {
	hm.debugLevel = level
}

// LogError used to log any error messages
func (hm *HM) LogError(f string, a ...interface{}) {
	if hm.debugLevel > 0 {
		hm.Log(f, a...)
	}
}

// LogInfo used to log any info messages
func (hm *HM) LogInfo(f string, a ...interface{}) {
	if hm.debugLevel > 1 {
		hm.Log(f, a...)
	}
}

// LogMsg used to log messages sent to and received from IPC layer
func (hm *HM) LogMsg(f string, a ...interface{}) {
	if hm.debugLevel > 2 {
		hm.Log(f, a...)
	}
}

//LogDebug used to log verbose debug info useful for debuggung the system
func (hm *HM) LogDebug(f string, a ...interface{}) {
	if hm.debugLevel > 3 {
		hm.Log(f, a...)
	}
}

// Log is called by all of the log functions and formats the messages and puts
// on the global Log channel
func (hm *HM) Log(f string, a ...interface{}) {
	s := fmt.Sprintf("[%d]-", hm.pid) + fmt.Sprintf(f, a...) + "\n"
	LogChan <- s
}

// Startup starts the HiveMind DSM on all processors. It should be the first call
func (hm *HM) Startup(gvec string) {

	var i uint8
	for i = 0; i < uint8(len(hm.locks)); i++ {
		hm.locks[i] = &Lock{
			mutex:   new(sync.Mutex),
			local:   hm.pid == hm.getLockManager(i),
			held:    false,
			lastReq: hm.getLockManager(i),
			nextID:  0,
			nextVC:  nil,
		}
	}

	if gvec != "" {
		process := gvec + strconv.Itoa(int(hm.pid))
		hm.vecLog = govec.InitGoVector(process, process)
	}
	// init barriers
	hm.barrierCnt = 0
	hm.halt = make(chan bool)

	locConf, err := configs.ReadConfig()
	if err != nil {
		//can't read from config
		return
	}
	fmt.Println("NRPROC: ", hm.nrProc, locConf, locConf.DroneList)

	dronesConnected := locConf.DroneList

	if locConf.IsCBM {
		//we're the first drone, start up the others
		_, dronesConnected, err := ipc.StartNodes(locConf.Drones[:hm.nrProc-1])
		if err != nil {
			//unable to start nodes sucessfully
		}
		//initialise IPC module
		hm.ipc, hm.txChan, hm.rxChan = ipc.Init(dronesConnected, "")
	} else {
		//not the first drone
		//initialise IPC module
		hm.ipc, hm.txChan, hm.rxChan = ipc.Init(dronesConnected, locConf.CBM)
	}
	// Start the background Receive handler
	go hm.rxMsgHandler()
}

// StartupTipc starts hivemind with test ipc library for unit testing
func (hm *HM) StartupTipc(port int, gvec string) {
	var err error
	hm.tipc, hm.rxChan, hm.txChan, err = tipc.NewConnection(port, hm.pid, hm.nrProc)
	if err != nil {
		fmt.Println("Startup Error: ", err)
		return
	}

	if gvec != "" {
		process := gvec + strconv.Itoa(int(hm.pid))
		hm.vecLog = govec.InitGoVector(process, process)
	}

	var i uint8
	for i = 0; i < uint8(len(hm.locks)); i++ {
		hm.locks[i] = &Lock{
			mutex:   new(sync.Mutex),
			local:   hm.pid == hm.getLockManager(i),
			held:    false,
			lastReq: hm.getLockManager(i),
			nextID:  0,
			nextVC:  nil,
		}
	}
	// init barriers
	hm.barrierCnt = 0
	hm.halt = make(chan bool)
	hm.LogInfo("VM Size=%d", hm.shm.GetSize())
	// Start the background Receive handler
	go hm.rxMsgHandler()
}

// ConnectToPeers is a test function to work with tipc for uniting testing
// hivemind. Will be replaced by ipc later
func (hm *HM) ConnectToPeers(ids []uint8, ips []string) {
	for i, id := range ids {
		if id != hm.pid {
			err := hm.tipc.Connect(ips[i], id)
			hm.LogDebug("Connecting to %d, err=%v\n", id, err)
		}
	}
}

// Exit is called to shutdown the HiveMind DSM.
func (hm *HM) Exit() {
	//hm.ipc.Close()
	elapsed := time.Since(hm.start)
	hm.LogInfo("Elapsed Time: %s", elapsed)
	//hm.Log("Page(%d):%s", 0, hm.shm.DumpPage(0))
}

//Read a single byte at the specified address
func (hm *HM) Read(addr int) (val byte, err error) {
	return hm.shm.Read(addr)
}

//ReadFloat reads a single float64 at the specified address
func (hm *HM) ReadFloat(addr int) (val float64, err error) {
	b, err := hm.shm.ReadN(addr, int(unsafe.Sizeof(float64(0))))
	if err != nil {
		return 0, err
	}
	bits := binary.BigEndian.Uint64(b)
	return math.Float64frombits(bits), nil
}

//ReadN an array of bytes at the specified address
func (hm *HM) ReadN(addr int, len int) (val []byte, err error) {
	return hm.shm.ReadN(addr, len)
}

//Write a byte at the specified address
func (hm *HM) Write(addr int, val byte) error {
	return hm.shm.Write(addr, val)
}

//WriteFloat writes a float at the specified address
func (hm *HM) WriteFloat(addr int, val float64) error {
	buf := make([]byte, int(unsafe.Sizeof(float64(0))))
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(val))
	return hm.shm.WriteN(addr, buf)
}

//WriteN an array of bytes at the specified address
func (hm *HM) WriteN(addr int, val []byte) error {
	return hm.shm.WriteN(addr, val)
}

//GetNProcs gets the number of processors configured for this DSM
func (hm *HM) GetNProcs() uint8 {
	return hm.nrProc
}

//GetProcID gets the processor ID of this process
func (hm *HM) GetProcID() uint8 {
	return hm.pid
}

// Barrier blocks the calling process until all other processes arrive at the barrier
func (hm *HM) Barrier(b uint8) {
	hm.LogDebug("Barrier(%d)..Start", b)
	manager := hm.getBarrierManager(b)
	// Only the non-manager processors send out the Barrier Request
	// to the manager. The manager is the only node receiving these
	// messages.
	if hm.pid != manager {
		hm.CreateInterval()
		lastLt := hm.procArray.GetLatestLocalTimestamp(manager)
		// get local intervals between lastLt and VC[pid]
		intervals := hm.procArray.GetProcMissingIntervals(hm.pid, lastLt)
		hm.IncorporateIntervals(intervals)
		hm.sendBarrierRequest(manager, b, &hm.VC, intervals)
	} else {
		intervals := make([]IntervalRec, 0)
		hm.sendBarrierRequest(manager, b, &hm.VC, intervals)
	}
	<-hm.waitChan
	hm.LogDebug("Barrier(%d)..Done", b)
}

// LockAcquire blocks the calling process until it acquires the specified lock
func (hm *HM) LockAcquire(lock uint8) {
	hm.LogDebug("LockAcq(%d)", lock)
	l := hm.locks[lock]
	l.mutex.Lock()
	if l.local {
		l.held = true
		l.mutex.Unlock()
	} else {
		hm.sendLockAcquireRequest(l.lastReq, lock)
		l.lastReq = hm.pid
		l.mutex.Unlock()
		<-hm.waitChan // Wait until we get the Grant
		hm.LogDebug("LockAcq(%d)..Granted", lock)
	}
}

// LockRelease releases the specified lock
func (hm *HM) LockRelease(lock uint8) {
	hm.LogDebug("LockRel(%d)", lock)
	l := hm.locks[lock]
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.held = false
	if l.nextVC != nil {
		hm.CreateInterval()
		hm.sendLockAcquireResponse(l.nextID, lock, l.nextVC)
		l.nextVC = nil
		l.nextID = hm.getLockManager(lock)
		l.local = false
		// l.lastReq =
	}
}

/*---------------------------------------------------------------------*/
/*------------------------Init Functions--------------------------*/
func (hm *HM) getLockManager(id uint8) uint8 {
	return id % hm.nrProc
}

func (hm *HM) getBarrierManager(id uint8) uint8 {
	return 0
}

// pageFaultHandler handles SEGV faults on the shared memory
func (hm *HM) pageFaultHandler(addr int, length int, fault shm.Prot) error {
	hm.LogDebug("SEGV[%d,%d] F=%d", addr, length, int(fault))

	pages, err := hm.shm.GetPages(addr, length)
	if err != nil {
		return err
	}
	protStats := hm.shm.GetProtStatusPages(pages)
	for i, page := range pages {
		pe := hm.pageArray.GetPageEntry(page)
		if protStats[i] == shm.PROT_NONE {
			if !pe.HasCopy() {
				hm.sendPageRequest(pe.GetManager(), int16(page))
				<-hm.waitChan
			}
			/*
				send (DiffRequest, pageId, diffId) to latest writers;
				receive (diffs) from latest writers;
				apply diffs to page in timestamp order;
			*/
			hm.sendDiffRequests(page)
			hm.applyDiffsInOrder(page)
		}
		prot := shm.PROT_READ
		if fault&shm.PROT_WRITE == shm.PROT_WRITE {
			twin := hm.shm.ReadPage(page)
			pe.SetTwin(twin)
			hm.dlMutex.Lock()
			hm.dirtyList[page] = true
			hm.dlMutex.Unlock()
			prot = shm.PROT_READ | shm.PROT_WRITE
		}
		pe.SetPageStatus(prot)
		hm.shm.MProtect(hm.pageArray.GetPageAddress(page), 1, prot)
	}
	return nil
}

// sendDiffRequests creates list of DiffRequest to get the latest updates for
// this page. It goes through each of the write notices and gets the minimum
// set of nodes to query
func (hm *HM) sendDiffRequests(page int) {
	hm.LogDebug("sendDiffReqs(%d)", page)

	type DiffRequestList struct {
		pid uint8
		req DiffRequest
	}

	diffRequests := make([]DiffRequestList, 0, hm.nrProc)
	var id uint8
	for id = 0; id < hm.nrProc; id++ {
		if id == hm.pid {
			continue
		}
		pe := hm.pageArray.GetPageEntry(page)
		start, end := pe.GetDiffRange(id)
		//hm.Log("DiffRange:S:%v,E:%v\n%s", start, end, pe.DumpWriteNotices())
		if end == nil {
			continue
		}

		updatedList := false
		for i, req := range diffRequests {
			if req.req.VcEnd.Covers(end) {
				// A request in the queue already covers this time interval.
				// Update the start time to be the lesser of the time in the
				// record and the time retrieved for this processor
				diffRequests[i].req.VcStart = *req.req.VcStart.Min(start)
				updatedList = true
			} else if end.Covers(&req.req.VcEnd) {
				// The new request timestamp covers this processor timestamp.
				// Adjust this record to go to this processor. Merge the end timestamps
				// and use min of both start time stamps
				diffRequests[i].pid = id
				diffRequests[i].req.VcStart = *req.req.VcStart.Min(start)
				diffRequests[i].req.VcEnd = *diffRequests[i].req.VcEnd.Merge(end)
				updatedList = true
			}
		}

		// The list was not updated. Add the request to the list
		if !updatedList {
			if start == nil {
				start = NewVectorClock(hm.nrProc)
			}
			req := DiffRequestList{
				pid: id,
				req: DiffRequest{
					Page:    int16(page),
					VcStart: *start,
					VcEnd:   *end,
				},
			}
			diffRequests = append(diffRequests, req)
		}
	}

	// Send the diffRequests
	for _, req := range diffRequests {
		hm.send(req.pid, DIFFREQ, req.req)
	}

	// Wait for all Diff Responses to be processed
	for range diffRequests {
		<-hm.waitChan
	}
}

// applyDiffsInOrder applies the received diffs in the order
// from earlies to latest diffs since the last time this
// function was called. The page array has the list of
// write notices for all processors. We also keep track of
// the last WN that was used to apply diff. Subsequent checks
// will start from this last count.
func (hm *HM) applyDiffsInOrder(page int) {
	hm.LogDebug("applyDiffs(%d)", page)
	pe := hm.pageArray.GetPageEntry(page)
	di := pe.diffIndex
	wnl := pe.writeNotices
	pdata := hm.shm.ReadPage(page)
	for {
		var vc *Vclock = nil
		var dp uint8 = 0
		var p uint8
		for p = 0; p < pe.nrProc; p++ {
			if len(wnl[p]) > di[p] {
				// New Write notice was added since last diff for this proc
				// Check if this can be used for the diff now
				wn := wnl[p][di[p]]
				if vc == nil || !wn.Vc.Covers(vc) {
					dp, vc = p, wn.Vc
				}
			}
		}

		// If vc is nil then we have completed all WN
		if vc == nil {
			break
		}

		// Apply the diff to the page
		if wnl[dp][di[dp]].Diff != nil {
			ApplyDiff(pdata, *wnl[dp][di[dp]].Diff)
		} else {
			//hm.LogDebug("ERR:Nil Diff found Page %d: [%d][%d] - %v", page, dp, di[dp], wnl[dp][di[dp]])
		}
		di[dp] += 1
	}

	// Write the buffer back to shm and save the diffIndex array
	hm.shm.WritePage(page, pdata)
	pe.diffIndex = di
}

/*---------------------------------------------------------------------*/
/*------------------------Messaging Functions--------------------------*/

// Send encodes the message buff and puts in on the IPC Tx channel
func (hm *HM) send(dest, msgID uint8, msg interface{}) {
	// First encode the message
	var buf bytes.Buffer
	e := gob.NewEncoder(&buf)
	if err := e.Encode(msg); err != nil {
		panic(err)
	}

	var mbuf []byte
	if hm.vecLog != nil {
		event := "Tx " + msgName[msgID]
		tmp := make([]byte, buf.Len())
		buf.Read(tmp[:])
		gobuf := hm.vecLog.PrepareSend(event, tmp)
		mbuf = make([]byte, 3+len(gobuf))
		mbuf[0], mbuf[1], mbuf[2] = dest, hm.pid, msgID
		copy(mbuf[3:], gobuf)
	} else {
		mbuf = make([]byte, 3+buf.Len())
		mbuf[0], mbuf[1], mbuf[2] = dest, hm.pid, msgID
		buf.Read(mbuf[3:])
	}
	// Prepend the dest id, msgType, and src id to the encoded gob
	hm.LogMsg("Send[%d]:Msg[%s],%v", dest, msgName[msgID], msg)
	hm.txChan <- mbuf
}

func (hm *HM) sendLockAcquireRequest(destID uint8, lockID uint8) {
	req := LockAcquireRequest{
		AcqID:  hm.pid,
		LockID: lockID,
		Vc:     *hm.VC.Copy(),
	}
	hm.send(destID, LOCKREQ, req)
}

func (hm *HM) forwardLockAcquireRequest(destID uint8, req *LockAcquireRequest) {
	hm.send(destID, LOCKREQ, req)
}

func (hm *HM) sendLockAcquireResponse(destID, lockID uint8, vc *Vclock) {
	intervals := hm.procArray.GetMissingIntervals(vc)
	rsp := LockAcquireResponse{
		LockID:    lockID,
		Vc:        *hm.VC.Copy(),
		Intervals: intervals,
	}
	hm.send(destID, LOCKRSP, rsp)
}

func (hm *HM) sendDiffResponse(destID uint8, page int16, wn []WriteNotice) {
	rsp := DiffResponse{
		Page: page,
		WN:   wn,
	}
	hm.send(destID, DIFFRSP, rsp)
}

func (hm *HM) sendPageRequest(destID uint8, page int16) {
	req := PageRequest{
		Page: page,
	}
	// Don't send copy request to yourself. Just change
	// the hasCopy flag to true and proceed
	if destID != hm.pid {
		hm.send(destID, PAGEREQ, req)
	} else {
		hm.pageArray.GetPageEntry(int(page)).SetHasCopy(true)
		hm.waitChan <- true
	}
}

func (hm *HM) sendPageResponse(destID uint8, page int16) {
	// Read the page from the twin copy if it exists else
	// get from memory
	pe := hm.pageArray.GetPageEntry(int(page))
	data := pe.GetTwin()
	if data == nil || len(data) == 0 {
		data = hm.shm.ReadPage(int(page))
	}

	rsp := PageResponse{
		Page:    page,
		Data:    data,
		CopySet: pe.GetCopySet(),
	}
	hm.send(destID, PAGERSP, rsp)
}

func (hm *HM) sendBarrierRequest(destID, b uint8, vc *Vclock, intervals []IntervalRec) {
	req := BarrierRequest{
		BarrierID: b,
		Vc:        *vc.Copy(),
		Intervals: intervals,
	}
	hm.send(destID, BARRREQ, req)
}

func (hm *HM) sendBarrierResponse(destID uint8, vc *Vclock) {
	intervals := hm.procArray.GetMissingIntervals(vc)
	rsp := BarrierResponse{
		Vc:        *hm.VC.Copy(),
		Intervals: intervals,
	}
	hm.send(destID, BARRRSP, rsp)
}

// RxMsgHandler is go routine whichHandles incoming messages from other drones
func (hm *HM) rxMsgHandler() {
	buf := bytes.NewBuffer([]byte{})
	for {
		var msgID, srcID uint8
		var mbuf []byte
		if hm.vecLog != nil {
			var gobuf []byte
			gobuf = <-hm.rxChan
			srcID, msgID = gobuf[1], gobuf[2]
			event := "Rx " + msgName[msgID]
			hm.vecLog.UnpackReceive(event, gobuf[3:], &mbuf)
			buf.Write(mbuf[:])
		} else {
			mbuf = <-hm.rxChan
			srcID, msgID = mbuf[1], mbuf[2]
			buf.Write(mbuf[3:])
		}

		d := gob.NewDecoder(buf)
		hm.LogMsg("Recv[%d]:Msg[%s]", srcID, msgName[msgID])
		switch msgID {
		case LOCKREQ:
			var r LockAcquireRequest
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handleLockAcquireRequest(srcID, &r)
		case LOCKRSP:
			var r LockAcquireResponse
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handleLockAcquireResponse(srcID, &r)
		case DIFFREQ:
			var r DiffRequest
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handleDiffRequest(srcID, &r)
		case DIFFRSP:
			var r DiffResponse
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handleDiffResponse(srcID, &r)
		case PAGEREQ:
			var r PageRequest
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handlePageRequest(srcID, &r)
		case PAGERSP:
			var r PageResponse
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handlePageResponse(srcID, &r)
		case BARRREQ:
			var r BarrierRequest
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handleBarrierRequest(srcID, &r)
		case BARRRSP:
			var r BarrierResponse
			if err := d.Decode(&r); err != nil {
				panic(err)
			}
			hm.handleBarrierResponse(srcID, &r)
		}
	}
}

func (hm *HM) handleLockAcquireRequest(srcID uint8, req *LockAcquireRequest) {
	lock := req.LockID
	l := hm.locks[lock]
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.held || (!l.local && l.lastReq == hm.pid) {
		if l.nextVC == nil {
			l.nextID = req.AcqID
			l.nextVC = &req.Vc
		} else {
			hm.forwardLockAcquireRequest(l.lastReq, req)
		}
	} else {
		if l.local {
			hm.CreateInterval()
			hm.sendLockAcquireResponse(req.AcqID, lock, &req.Vc)
			l.local = false
		} else {
			hm.forwardLockAcquireRequest(l.lastReq, req)
		}
	}
	l.lastReq = req.AcqID
}

func (hm *HM) handleLockAcquireResponse(srcID uint8, rsp *LockAcquireResponse) {
	lock := rsp.LockID
	l := hm.locks[lock]
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Do we need to create a new interval?? We already did this at end of request

	hm.IncorporateIntervals(rsp.Intervals)
	l.held = true
	l.local = true
	hm.waitChan <- true
}

func (hm *HM) handleDiffRequest(srcID uint8, req *DiffRequest) {

	currentInterval := len(hm.procArray.pa[hm.pid]) - 1
	currentIntervalRecord := hm.procArray.pa[hm.pid][currentInterval]

	// If the Twin exist for ths page then we create a new write notice,
	// generate the diff and delete the twin
	page := int(req.Page)
	pageEntry := hm.pageArray.GetPageEntry(page)
	twin := pageEntry.GetTwin()
	if twin != nil && len(twin) > 0 {
		data := hm.shm.ReadPage(page)
		diff := GetDiff(twin, data)
		pageEntry.DelTwin()
		hm.shm.MProtect(hm.pageArray.GetPageAddress(page), 1, shm.PROT_READ)
		pageEntry.SetPageStatus(shm.PROT_READ)
		if diff != nil {
			// If the page is dirty them we need to create a new WN record
			hm.dlMutex.Lock()
			if hm.dirtyList[page] {
				wn := NewWriteNotice(page, currentIntervalRecord)
				pageEntry.AddWN(wn.ProcID, wn)
			}
			hm.dlMutex.Unlock()
			for _, wn := range currentIntervalRecord.WriteNotices {
				if wn.GetPageID() == page {
					wn.SetDiff(&diff)
				}
			}
		}
	}

	var i uint8
	writeNotices := make([]WriteNotice, 0)
	for i = 0; i < hm.nrProc; i++ {
		if i == srcID {
			continue
		}
		wnl := pageEntry.writeNotices[i]
		for w := len(wnl) - 1; w >= 0; w-- {
			if !wnl[w].Vc.Covers(&req.VcStart) {
				break
			} else if req.VcEnd.Covers(wnl[w].Vc) {
				writeNotices = append(writeNotices, *wnl[w])
			}
		}
	}

	hm.sendDiffResponse(srcID, req.Page, writeNotices)

}

func (hm *HM) handleDiffResponse(srcID uint8, rsp *DiffResponse) {
	for _, wn := range rsp.WN {
		for _, ir := range hm.procArray.pa[wn.ProcID] {
			for _, wn2 := range ir.GetWN() {
				vc1 := wn.GetTS()
				vc2 := ir.GetTS()
				if vc2.IsIdentical(vc1) && wn2.PageID == int(rsp.Page) {
					wn2.SetDiff(wn.GetDiff())
					break
				}
			}
		}
	}
	hm.waitChan <- true
}

func (hm *HM) handlePageRequest(srcID uint8, req *PageRequest) {
	/*
		receive (PageRequest, pageId) from req node;
		if (copyset={pid} and page status=read-write)
		        set page status to read-only;
		copyset := copyset union {req};
		send (page copy, copyset) to req node;
	*/

	hm.sendPageResponse(srcID, req.Page)
	pe := hm.pageArray.GetPageEntry(int(req.Page))
	pe.AddCopySet(srcID)
}

func (hm *HM) handlePageResponse(srcID uint8, rsp *PageResponse) {
	hm.shm.WritePage(int(rsp.Page), rsp.Data)
	pe := hm.pageArray.GetPageEntry(int(rsp.Page))
	pe.SetHasCopy(true)
	for _, id := range rsp.CopySet {
		pe.AddCopySet(id)
	}
	hm.waitChan <- true
}

// handleBarrierRequest handles incoming barrier requests. Note that only
// the Manager for the barrier receives requests. The manager waits until
// receives request from all clients before proceeding. Therefore, it saves
// each request in the in the barrierReq queue.
func (hm *HM) handleBarrierRequest(srcID uint8, req *BarrierRequest) {

	// Save request for this srcID
	hm.barrierReq[srcID] = req

	// Increment the count and return if all messages have not been received
	hm.barrierCnt++
	if hm.barrierCnt < hm.nrProc {
		return
	}

	// We got all requests. First process the requests and then send responses
	// back to each client. Don't send response to the manager.
	hm.LogDebug("Barrier Received Reqs: %v", hm.VC)
	lastLt := *hm.VC.Copy()
	hm.CreateInterval()
	intervals := hm.procArray.GetProcMissingIntervals(hm.pid, &lastLt)
	hm.IncorporateIntervals(intervals)

	for p, req := range hm.barrierReq {
		if uint8(p) != hm.pid {
			hm.IncorporateIntervals(req.Intervals)
		}
	}
	var i uint8
	for i = 0; i < hm.nrProc; i++ {
		if i != hm.pid {
			hm.sendBarrierResponse(i, &hm.barrierReq[i].Vc)
		}
	}
	hm.barrierCnt = 0
	hm.waitChan <- true
}

func (hm *HM) handleBarrierResponse(srcID uint8, rsp *BarrierResponse) {
	hm.IncorporateIntervals(rsp.Intervals)
	hm.waitChan <- true
}

// CreateInterval creates a new interval record for the current interval;
// and for all the pages modified during the interval it creates write
// notice records.
func (hm *HM) CreateInterval() {
	/*
		if(dirtyList is not empty) {
		        VC[pid] := VC[pid] + 1;
		        insert new interval i in procArray[pid];
		        for each page in dirtyList
		                create write notice record for interval i;
		        clear dirtyList;
		}
	*/
	if len(hm.dirtyList) == 0 {
		return
	}

	hm.VC.Increment(hm.pid)
	interval := NewIntervalRecord(hm.pid, hm.VC.Copy())
	hm.procArray.InsertInterval(hm.pid, interval)
	hm.dlMutex.Lock()
	for pg := range hm.dirtyList {
		wn := NewWriteNotice(pg, interval)
		hm.pageArray.GetPageEntry(pg).AddWN(wn.ProcID, wn)
	}
	hm.dirtyList = make(map[int]bool, 0)
	hm.dlMutex.Unlock()

	//hm.LogDebug("CI:Proc Array:\n%s", hm.procArray.Dump())
	//hm.LogDebug("CI:Page Array:\n%s", hm.pageArray.GetPageEntry(0).DumpWriteNotices())
}

// IncorporateIntervals is called by the lock acquirer to incorporate the
// information received in the reply in its own data structures. Notice
// that the system creates a diff for writable pages for which a write
// notice was received. This is important to allow the system to
// distinguish the modifications made by concurrent writers to the page.
// The status of the pages for which write notices were received in the
// reply is set to no-access. When the acquirer tries to access one of
// the invalidated pages the system invokes the PageFaultHandler
func (hm *HM) IncorporateIntervals(intervals []IntervalRec) {
	/*
			for each i in intervals {
		        insert record for i in procArray[i.idCreat];
		        VC[i.idCreat] := i.vc[i.idCreat];
		        for each write notice in i {
		          store write notice;
		          if (twin exists for the page write notice refers to) {
						if(a write notice corresponding to the current
							writes was not already created)
		                        CreateInterval;
		                create diff;
		                delete twin;
		          }
		          set page status to no-access;
				}

	*/
	//hm.Log("II:Intervals:\n%v", intervals)
	for _, irec := range intervals {
		id := irec.ProcID

		i := hm.procArray.GetLastRecord(hm.pid)
		if id != hm.pid {
			i = hm.procArray.InsertIntervalRec(hm, irec)
			intervalVC, _ := i.GetTS().GetInterval(id)
			hm.VC.Update(id, intervalVC)
		}

		if i == nil {
			return
		}

		for _, wn := range i.GetWN() {
			// store the write notices with the interval record. Should
			// already be linked to the record
			//hm.Log("Proc Array:%p\n%s", wn, hm.procArray.Dump())
			pageID := wn.GetPageID()
			pageEntry := hm.pageArray.GetPageEntry(pageID)
			twin := pageEntry.GetTwin()
			if twin != nil && len(twin) > 0 {
				page := hm.shm.ReadPage(pageID)
				diff := GetDiff(twin, page)
				wn.SetDiff(&diff)
				pageEntry.DelTwin()
			}
			hm.shm.MProtect(hm.pageArray.GetPageAddress(pageID), 1, shm.PROT_NONE)
			pageEntry.SetPageStatus(shm.PROT_NONE)
		}
	}
	//hm.LogDebug("II:Proc Array:\n%s", hm.procArray.Dump())
	//hm.LogDebug("II:Page Array:\n%s", hm.pageArray.GetPageEntry(0).DumpWriteNotices())
}

func DumpLog() {
	for s := range LogChan {
		fmt.Print(s)
	}
}
