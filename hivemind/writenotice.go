/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the WriteNoticeRecord
*/
package hivemind

import (
	"fmt"
)

// DiffInfo struct for communicating diffs between drones
type DiffInfo struct {
	Diff     Diff
	ProcID   uint8
	Interval uint
	PageID   int
}

// DiffChar keeps track of one character difference in a page
type DiffChar struct {
	Offset int  // offset into the page
	Char   byte // new value at the offset
}

// Diff is the data structure that describes the diff between two pages
type Diff []DiffChar

// WriteNoticeIf defines methods supported by write notice records
type WriteNoticeIf interface {
	//GetInterval gets the interval record
	GetInterval() *IntervalRecord

	// GetDiff gets the pointer to the run-length encode page diff
	GetDiff() *Diff

	// SetDiff sets the diff in this record
	SetDiff(diff *Diff)

	// GetPageID gets the pageID of the record
	GetPageID() int
}

// WriteNotice describes modifications to a page. Instead of including a
// link to the interval record we just put the timestamp and procID. It
// simplifies the messaging since we can use this structure in the
// message requests/responses
type WriteNotice struct {
	ProcID uint8   // Creator of this Write Notice
	Diff   *Diff   // pointer to diff
	Vc     *Vclock // pointer to the vector time when created
	PageID int     // number of this page
}

// NewWriteNotice creates a new Write Notice record using the parameters from
// the interval. The write notice is added to Page Array and to the Interval
func NewWriteNotice(page int, interval *IntervalRecord) *WriteNotice {
	wn := new(WriteNotice)
	wn.PageID = page
	wn.Vc = interval.Vc.Copy()
	wn.ProcID = interval.ProcID
	wn.Diff = nil
	//hm.pageArray.GetPageEntry(page).AddWN(wn.ProcID, wn)
	interval.AddWN(wn)
	return wn
}

//GetInterval gets the interval record
func (wn *WriteNotice) GetTS() *Vclock {
	return wn.Vc

}

// GetDiff gets the pointer to the run-length encode page diff
func (wn *WriteNotice) GetDiff() *Diff {
	return wn.Diff
}

// SetDiff sets the diff in this record
func (wn *WriteNotice) SetDiff(diff *Diff) {
	wn.Diff = diff
}

// SetDiff sets the diff in this record
func (wn *WriteNotice) EmptyDiff() bool {
	return !(wn.Diff != nil && len(*wn.Diff) > 0)
}

// GetPageID gets the pageID of the record
func (wn *WriteNotice) GetPageID() int {
	return wn.PageID
}

func (wn WriteNotice) String() string {
	return fmt.Sprintf("WN[P%d,p%d,VC%v,D(%v)]", wn.ProcID, wn.PageID, wn.Vc, wn.Diff)
}

// GetDiff generates the run-length encode diff between two pages
func GetDiff(old, new []byte) Diff {
	// Both pages should match
	if len(old) != len(new) {
		return nil
	}

	diff := make([]DiffChar, 0)
	for i, v := range old {
		if v != new[i] {
			diff = append(diff, DiffChar{i, new[i]})
		}
	}
	return diff
}

// ApplyDiff updates the page with the diffs
func ApplyDiff(page []byte, diff Diff) {
	pglen := len(page)
	for _, v := range diff {
		if v.Offset < pglen {
			page[v.Offset] = v.Char
		} else {
			fmt.Println("Exceeded end of page")
		}
	}
}
