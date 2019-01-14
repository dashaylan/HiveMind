/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the IntervalRecord
*/
package hivemind

import "fmt"

// IntervalRecordIf defines the interface for InterfaceRecord types
type IntervalRecordIf interface {
	//GetID gets the procID of the record
	GetID() uint8

	// GetTS gets the vector timestamp
	GetTS() Vclock

	//GetWN gets the list Write Notice pointers in the interval
	GetWN() []*WriteNotice

	//AddWN adds the WriteNotice to the record
	AddWN(wn *WriteNotice)

	//DelWN deletes the WriteNotice from the record
	DelWN(wn *WriteNotice)
}

// IntervalRecord describes the interval
type IntervalRecord struct {
	ProcID       uint8          // id of process which created the interval
	Vc           *Vclock        // vector time of creator
	WriteNotices []*WriteNotice // list of write notice records for this interval.
}

//NewIntervalRecord is constructor for an interval record
func NewIntervalRecord(pid uint8, vc *Vclock) *IntervalRecord {
	ir := new(IntervalRecord)
	ir.ProcID = pid
	ir.Vc = vc.Copy()
	ir.WriteNotices = make([]*WriteNotice, 0)
	return ir
}

func (i IntervalRecord) String() string {
	return fmt.Sprintf("(%d,%v,%v)", i.ProcID, i.Vc, i.WriteNotices)
}

//GetID gets the procID of the record
func (i *IntervalRecord) GetID() uint8 {
	return i.ProcID
}

// GetTS gets the vector timestamp
func (i *IntervalRecord) GetTS() *Vclock {
	return i.Vc
}

//GetWN gets the list Write Notice pointers in the interval
func (i *IntervalRecord) GetWN() []*WriteNotice {
	return i.WriteNotices
}

//AddWN adds the WriteNotice to the record
func (i *IntervalRecord) AddWN(wn *WriteNotice) {
	for _, ir := range i.WriteNotices {
		if ir == wn {
			return
		}
	}
	i.WriteNotices = append(i.WriteNotices, wn)
}

//DelWN deletes the WriteNotice from the record
func (i *IntervalRecord) DelWN(wn *WriteNotice) {
	for j, ir := range i.WriteNotices {
		if ir == wn {
			i.WriteNotices = append(i.WriteNotices[:j], i.WriteNotices[j+1:]...)
		}
	}

}
