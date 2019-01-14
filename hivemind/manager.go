/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the Barrier and Lock managers
*/
package hivemind

import (
	"../ipc"
	"sync"
)

// Manager manages the DSM processes
type Manager interface {
	getPID() int
}


type MD struct {
	pid  int
}


func (md *MD) GetPID() int {
	return md.pid
}

var peers ManaStruct

//when all of tracker = curPhase, we know all
//drones have reached barrier. then we simply
//flip curPhase
type ManaStruct struct {
	mut      *sync.Mutex
	tracker  map[string]bool
	curPhase bool
	length   int
	counter  int
	Txchan   chan ipc.IpcMessage
	him *HM
}

func (ms *ManaStruct) Init(keylist []string, himpt *HM) {
	ms.mut = new(sync.Mutex)
	ms.curPhase = true
	ms.counter = 0
	ms.length = len(keylist)
	//ms.Txchan = Tx
	ms.him = himpt
	for _, key := range keylist {
		ms.tracker[key] = false
	}
}

func (ms *ManaStruct) SetBar(drone string, message BarrierRequest) {
	ms.mut.Lock()
	stat, ex := ms.tracker[drone]
	if ex {
		if stat != ms.curPhase {
			ms.tracker[drone] = ms.curPhase
			ms.counter++
		}
	}
	if ms.counter == ms.length {
		ms.mut.Unlock()
		ms.Broadcast()
	} else {
		ms.mut.Unlock()
	}
}

//all drones have reached barrier
func (ms *ManaStruct) Broadcast() {
	ms.mut.Lock()
	defer ms.mut.Unlock()
	ms.counter = 0
	ms.curPhase = !ms.curPhase
	for dronekey := range ms.tracker {
		go func(drone string) {
			//TODO form barrier message
			//barmes := BarrierResponse{ms.him.VC}
			//ms.him.send(drone, BARRRSP, barmes)
		}(dronekey)
	}
}

func InitManager(initList []string, Tx chan ipc.IpcMessage) {
	peers = ManaStruct{}
	//peers.Init(initList, Tx)
}

func BarrierCall(caller string) {
	//peers.SetBar(caller)
}
