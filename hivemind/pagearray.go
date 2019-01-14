/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the PageArray and PageEntry objects
*/
package hivemind

import (
	"fmt"
	"sync"

	"../shm"
)

// PageEntry maintains the stats of shared page in the DSM
type PageEntry struct {
	pageID       int              // number of this page
	pid          uint8            // processor id
	nrProc       uint8            // Number of processors
	status       shm.Prot         // operating system protection status for page
	twin         []byte           // original copy of the page
	writeNotices [][]*WriteNotice // array with one list of write notice records per process
	manager      uint8            // identification of the page manager
	copyset      []uint8          // set of processes with copy of the page
	hasCopy      bool             // true if this processor has copy of the page in SHM
	mutex        *sync.RWMutex    // mutex to protect multiple rw to this struct
	diffIndex    []int            // Keeps track of the last diff applied for this processor

}

//
// Page Entry Methods
//
func NewPageEntry(pid uint8, page int, nrProc uint8) *PageEntry {
	pe := new(PageEntry)
	pe.pid = pid
	pe.pageID = page
	pe.nrProc = nrProc
	pe.status = shm.PROT_NONE
	pe.hasCopy = false
	pe.copyset = make([]uint8, 0)
	pe.mutex = new(sync.RWMutex)
	pe.diffIndex = make([]int, nrProc)
	pe.writeNotices = make([][]*WriteNotice, nrProc)
	for i := range pe.writeNotices {
		pe.writeNotices[i] = make([]*WriteNotice, 0)
	}
	return pe
}

func (pe *PageEntry) GetPageID() int {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	return pe.pageID
}

func (pe *PageEntry) GetPageStatus() shm.Prot {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	return pe.status
}

func (pe *PageEntry) SetPageStatus(prot shm.Prot) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	pe.status = prot
}

func (pe *PageEntry) DelTwin() {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	pe.twin = make([]byte, 0)
}

func (pe *PageEntry) GetTwin() []byte {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	return pe.twin
}

func (pe *PageEntry) SetTwin(twin []byte) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	pe.twin = make([]byte, len(twin))
	copy(pe.twin, twin)
}

func (pe *PageEntry) HasCopy() bool {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	return pe.hasCopy
}

func (pe *PageEntry) SetHasCopy(h bool) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	pe.hasCopy = h
}

func (pe *PageEntry) GetCopySet() []uint8 {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	copyset := make([]uint8, len(pe.copyset))
	copy(copyset, pe.copyset)
	return copyset
}

func (pe *PageEntry) AddCopySet(proc uint8) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	for _, p := range pe.copyset {
		if p == proc {
			return
		}
	}
	pe.copyset = append(pe.copyset, proc)
}

func (pe *PageEntry) SetManager(m uint8) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	pe.manager = m
}

func (pe *PageEntry) GetManager() uint8 {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	return pe.manager
}

func (pe *PageEntry) AddWN(proc uint8, wn *WriteNotice) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	if wn != nil {
		pe.writeNotices[proc] = append(pe.writeNotices[proc], wn)
	}
}

func (pe *PageEntry) GetDiffRange(proc uint8) (start *Vclock, end *Vclock) {
	// Get the first and last time assuming that the times are in order
	// from earliest to latest
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	wnList := pe.writeNotices[proc]
	i := len(wnList) - 1
	start, end = nil, nil
	if i >= 0 && wnList[i].EmptyDiff() {
		end = wnList[i].GetTS().Copy()
		for ; i >= 0; i-- {
			if wnList[i].EmptyDiff() {
				break
			}
			start = wnList[i].GetTS().Copy()
		}
	}
	return start, end
}

func (pe *PageEntry) DumpWriteNotices() string {
	var s string
	for p := 0; p < int(pe.nrProc); p++ {
		s += fmt.Sprintf("WN-P[%d]:", p)
		for _, w := range pe.writeNotices[p] {
			s += fmt.Sprintf(" %v,", w)
		}
		s += "\n"
	}
	return s
}

// PageArray is an array of PageEntries -- one  for each processor
type PageArray struct {
	pga      []*PageEntry
	pageSize int
	pid      uint8
	nrProc   uint8
}

// NewPageArray creates a new instance of the page array
func NewPageArray(nrPages int, pageSize int, procId, nrProc uint8) *PageArray {
	pgArray := new(PageArray)
	pgArray.pageSize = pageSize
	pgArray.pid = procId
	pgArray.nrProc = nrProc
	pgArray.pga = make([]*PageEntry, nrPages)
	for i := 0; i < nrPages; i++ {
		pgArray.pga[i] = NewPageEntry(procId, i, nrProc)
	}
	return pgArray
}

func (pga *PageArray) GetPageSize() int {
	return pga.pageSize
}

func (pga *PageArray) GetPageAddress(page int) int {
	return pga.pageSize * page
}

func (pga *PageArray) GetPageEntry(page int) *PageEntry {
	return pga.pga[page]
}
