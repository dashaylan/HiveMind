/*
Package hivemind implements the TreadMarks API.

This file contains the unit tests for the API and the
top level handlers
*/
package hivemind

import (
	"testing"
)

const (
	PID      = 0
	NRPROC   = 3
	NRBARR   = 1
	NRLOCKS  = 10
	NRPAGES  = 5
	PAGESIZE = 30
)

var PROCADDR = []string{"111.1", "111.2", "111.3"}

func TestHMCreation(t *testing.T) {
	hm := NewHiveMind(PID, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, PROCADDR)
	hm.Startup()

	if hm.nrLocks != NRLOCKS {
		t.Errorf("Expected HiveMind to track %d locks, has %d", NRLOCKS, hm.nrLocks)
	}

	if hm.nrBarriers != NRBARR {
		t.Errorf("Expected HiveMind to have %d barriers, has %d", NRBARR, hm.nrBarriers)
	}

	if len(hm.locks) != NRLOCKS {
		t.Errorf("Expected HiveMind to create %d locks, has %d", NRLOCKS, len(hm.locks))
	}

	if hm.shm.GetSize() != NRPAGES*PAGESIZE {
		t.Errorf("Expected HiveMind to have shm with size of %d, has %d", NRPAGES*PAGESIZE, hm.shm.GetSize())
	}

	if len(hm.pageArray.pga) != NRPAGES {
		t.Errorf("Expected HiveMind to have PageArray with %d pages, has %d", NRPAGES, len(hm.pageArray.pga))
	}

	if len(hm.procArray.pa) != NRPROC {
		t.Errorf("Expected HiveMind to have a ProcArray with %d drones, has %d", NRPROC, len(hm.procArray.pa))
	}

	if len(hm.VC.ClockMap) != NRPROC {
		t.Errorf("Expected HiveMind to have a VectorClock with %d times, has %d", NRPROC, len(hm.VC.ClockMap))
	}

}

func TestLockAcquire(t *testing.T) {
	/*
		hm := NewHiveMind(PID, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, PROCADDR)
		hm.Startup()
		fmt.Println("info")
		fmt.Println(hm.locks[1])
		fmt.Println(hm.procArray.pa[0])
		hm.LockAcquire(2)
	*/
}
