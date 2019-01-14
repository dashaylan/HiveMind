/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the vector timestamp
*/
package hivemind

import (
	"errors"
	"fmt"
)

// Vclock Struct which includes the map that keeps track of intervals
type Vclock struct {
	ClockMap []uint
}

// Get new vector clock that does not contain any values
func NewVectorClock(nrProc uint8) *Vclock {
	vc := new(Vclock)
	vc.ClockMap = make([]uint, nrProc)
	return vc
}

// The following APIs are needed by TreadMarks
/*
- increment(clock, nodeId): increment a vector clock at "nodeId"
func (vc *Vclock) Increment(nodeId int)
- merge(a, b): given two vector clocks, returns a new vector clock with all values greater than those of the merged clocks
func (vc *Vclock) Merge(b Vclock) VClock
- compare(a, b): compare two vector clocks, returns -1 for a < b and 1 for a > b; 0 for concurrent and identical values.
func (vc *Vclock) Compare(b Vclock) int
- isConcurrent(a, b): if A and B are equal, or if they occurred concurrently.
func (vc *Vclock) IsConcurrent(b Vclock) bool
- isIdentical(a, b): if every value in both vector clocks is equal.
func (vc *Vclock) IsEqual(b Vclock) bool
*/

// Updates the vector clock, if the clock does not contain
// the given id then it creates a new entry for that id
func (vc *Vclock) Update(id uint8, time uint) {
	vc.ClockMap[id] = time
}

// GetInterval Gets the interval for the given id
func (vc *Vclock) GetInterval(id uint8) (uint, error) {
	if id < 0 || id > uint8(len(vc.ClockMap))-1 {
		return 0, errors.New(fmt.Sprintf("Invalid ID: %d", id))
	}

	return vc.ClockMap[id], nil
}

// Merge Merges two vector clocks together taking the pairwise between
// two intervals with the same id
func (vc *Vclock) Merge(vc2 *Vclock) *Vclock {
	vcm := NewVectorClock(uint8(len(vc.ClockMap)))
	for id, interval := range vc2.ClockMap {

		if interval > vc.ClockMap[id] {
			vcm.ClockMap[id] = interval
		} else {
			vcm.ClockMap[id] = vc.ClockMap[id]
		}
	}
	return vcm
}

// Covers checks the VC covers another VC according to the covered
// definition described in Keleher.
func (vc *Vclock) Covers(vc2 *Vclock) bool {
	for i, val := range vc.ClockMap {
		if vc2.ClockMap[i] > val {
			return false
		}
	}
	return true
}

func (vc *Vclock) Min(vc2 *Vclock) *Vclock {
	vcmin := NewVectorClock(uint8(len(vc.ClockMap)))
	if vc2 == nil {
		return vcmin
	}
	for i := range vcmin.ClockMap {
		if vc.ClockMap[i] < vc2.ClockMap[i] {
			vcmin.ClockMap[i] = vc.ClockMap[i]
		} else {
			vcmin.ClockMap[i] = vc2.ClockMap[i]
		}
	}
	return vcmin
}

// Copy Copies contents of a vector clock into a new vector clock struct
func (vc *Vclock) Copy() *Vclock {
	vc2 := new(Vclock)
	vc2.ClockMap = make([]uint, len(vc.ClockMap))

	for id, interval := range vc.ClockMap {
		vc2.ClockMap[id] = interval
	}

	return vc2
}

// Increment Increments value given by id and returns the incremented value
func (vc *Vclock) Increment(id uint8) (uint, error) {
	if id < 0 || id > uint8(len(vc.ClockMap)) {
		return 0, fmt.Errorf("Invalid ID: %d", id)
	}

	vc.ClockMap[id] = vc.ClockMap[id] + 1
	return vc.ClockMap[id], nil
}

// Compare Returns a slice of ids which have interval values less than
// the current vector clock or are not present in the vector clock being
// compared to but present in the current vector clock
// Implementation may change
func (vc *Vclock) Compare(vc2 *Vclock) []int {
	comparisons := make([]int, len(vc.ClockMap))
	for id, interval := range vc.ClockMap {

		if interval < vc2.ClockMap[id] {
			comparisons[id] = -1
		} else if interval > vc2.ClockMap[id] {
			comparisons[id] = 1
		} else {
			comparisons[id] = 0
		}
	}
	return comparisons
}

// IsIdentical Checks if two vector clocks are identical, i.e. they have
// the same value for all processes
func (vc *Vclock) IsIdentical(vc2 *Vclock) bool {
	compareArr := vc.Compare(vc2)

	for _, compareVal := range compareArr {
		if compareVal != 0 {
			return false
		}
	}

	return true
}

func (vc Vclock) String() string {
	return fmt.Sprintf("%v", vc.ClockMap)
}

// VCToArray Returns internal clockmap uint slice
func VCToArray(vc *Vclock) []uint {
	return vc.ClockMap
}

// ArrayToVC Creates a new vc from an uint slice
func ArrayToVC(arr []uint) *Vclock {
	vc := new(Vclock)
	vc.ClockMap = arr

	return vc
}
