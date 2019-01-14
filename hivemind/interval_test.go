/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the unit tests for IntervalRecord
*/
package hivemind

import "testing"

func TestNewIntervalRecord(t *testing.T) {
	ir := NewIntervalRecord(1, NewVectorClock(1))

	if ir == nil {
		t.Errorf("[TEST] Interval record is nil")
	}
}

func TestIntervalGetID(t *testing.T) {
	ir := NewIntervalRecord(1, NewVectorClock(1))

	if ir.GetID() != 1 {
		t.Errorf("[TEST] Invalid vector clock value for interval record got %d expected %d", ir.GetID(), 1)
	}
}

func TestIntervalAddGetDelWN(t *testing.T) {
	ir := NewIntervalRecord(1, NewVectorClock(1))

	b1 := []byte{0xA1, 0xB2, 0xC3}
	b2 := []byte{0xA0, 0xB2, 0xC4}

	diff := GetDiff(b1, b2)

	wn := new(WriteNotice)

	wn.Diff = &diff
	wn.PageID = 3

	ir.AddWN(wn)

	wn1 := ir.GetWN()[0]

	if wn1.Diff != &diff {
		t.Errorf("[TEST] Invalid interval record write notice")
	}

	ir.DelWN(wn)

	if len(ir.GetWN()) != 0 {
		t.Errorf("[TEST] Invalid interval record write notice list length got %d expected %d", len(ir.GetWN()), 0)
	}

}

func TestIntervalGetTS(t *testing.T) {
	ir := NewIntervalRecord(1, NewVectorClock(1))

	ir.GetTS().Update(0, 1)

	interval, err := ir.GetTS().GetInterval(0)

	if err != nil {
		t.Errorf("[TEST] Error access interval record vector clock %s", err.Error())
	}

	if interval != 1 {
		t.Errorf("[TEST] Invalid vector clock value for interval record got %d expected %d", interval, 1)
	}
}
