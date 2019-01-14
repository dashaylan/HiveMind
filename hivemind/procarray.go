/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the processor array and
processor entry objects
*/
package hivemind

import (
	"fmt"
	"sync"
)

// ProcArray maintains a list of interval records for each processor
type ProcArray struct {
	pa     [][]*IntervalRecord // first index is processor
	nrProc uint8               // number of processors
	count  int                 // number of intervals inserted
	mutex  *sync.RWMutex       // Mutex to protect read/writes to procArray

}

// NewProcArray creates a new instance of the procArray
func NewProcArray(nrProc uint8) *ProcArray {
	p := new(ProcArray)
	p.pa = make([][]*IntervalRecord, nrProc)
	p.nrProc = nrProc
	p.mutex = new(sync.RWMutex)
	return p
}

// InsertInterval inserts an interval to the end of the processor list
func (pa *ProcArray) InsertInterval(pid uint8, interval *IntervalRecord) {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()
	pa.pa[pid] = append(pa.pa[pid], interval)
	pa.count++
}

// InsertIntervalRec inserts an new IntervalRecord from a IntervalRec. It also creates
// new writes notices for each of the pages in the Interval pages. The WN are added
// to the new interval record and also added to the page array. The Interval Record is
// then appended to the procArray
func (pa *ProcArray) InsertIntervalRec(hm *HM, interval IntervalRec) *IntervalRecord {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()
	ir := NewIntervalRecord(interval.ProcID, &interval.Vc)
	for _, p := range interval.Pages {
		wn := NewWriteNotice(p, ir)
		hm.pageArray.GetPageEntry(p).AddWN(wn.ProcID, wn)
	}
	pa.pa[interval.ProcID] = append(pa.pa[interval.ProcID], ir)
	pa.count++
	return ir
}

// GetLatestLocalTimestamp gets the latest local timestamp known for a specific processor
func (pa *ProcArray) GetLatestLocalTimestamp(pid uint8) *Vclock {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()
	l := len(pa.pa[pid])
	if l == 0 {
		return NewVectorClock(pa.nrProc)
	}
	return pa.pa[pid][l-1].Vc.Copy()
}

func getIntervalRec(ir IntervalRecord) IntervalRec {
	pages := make([]int, len(ir.WriteNotices))
	for i := 0; i < len(pages); i++ {
		pages[i] = ir.WriteNotices[i].GetPageID()
	}
	return IntervalRec{
		ProcID: ir.ProcID,
		Vc:     *ir.Vc.Copy(),
		Pages:  pages,
	}
}

// GetProcMissingIntervals gets the intervals on a processor that is not covered by TS
func (pa *ProcArray) GetProcMissingIntervals(pid uint8, vc *Vclock) []IntervalRec {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()
	pi := pa.pa[pid]
	intervals := make([]IntervalRec, 0, len(pi))
	for i := len(pi) - 1; i >= 0; i-- {
		if vc.Covers(pi[i].Vc) {
			break
		}
		intervals = append(intervals, getIntervalRec(*pi[i]))
	}
	return intervals
}

// GetMissingIntervals gets the intervals on all processors not covered by TS
func (pa *ProcArray) GetMissingIntervals(vc *Vclock) []IntervalRec {
	intervals := make([]IntervalRec, 0, pa.count)
	var pr uint8
	for pr = 0; pr < pa.nrProc; pr++ {
		intervals = append(intervals, pa.GetProcMissingIntervals(pr, vc)...)
	}
	return intervals
}

// Dump returns the processor contents as a string
func (pa *ProcArray) Dump() string {
	var pr uint8
	var s string
	for pr = 0; pr < pa.nrProc; pr++ {
		s += fmt.Sprintln("PR[", pr, "]", pa.pa[pr])
	}
	return s
}

//GetLastRecord gets the latest interval record for a processor
func (pa *ProcArray) GetLastRecord(p uint8) *IntervalRecord {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()
	l := len(pa.pa[p])
	if l > 0 {
		return pa.pa[p][l-1]
	}
	return nil
}
