/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the unit tests for the
vector timestamp
*/
package hivemind

import (
	"testing"
)

func TestCreateUpdateIncrementVectorClock(t *testing.T) {
	vectorClock := NewVectorClock(1)

	vectorClock.Update(0, 10)

	interval, err := vectorClock.GetInterval(0)

	if err != nil {
		t.Errorf("[TEST] Can't update vector clock: %s", err.Error())
	}

	if interval != 10 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", interval, 10)
	}

	vectorClock.Update(0, 0)

	interval, err = vectorClock.GetInterval(0)

	if err != nil {
		t.Errorf("[TEST] VectorClockUpdate: Can't update vector clock: %s", err.Error())
	}

	if interval != 0 {
		t.Errorf("[TEST] VectorClockUpdate: Invalid vector clock value got %d expected %d", interval, 0)
	}

	interval, err = vectorClock.Increment(0)

	if err != nil {
		t.Errorf("[TEST] VectorClockUpdate: Can't update vector clock %s", err.Error())
	}

	if interval != 1 {
		t.Errorf("[TEST] VectorClockUpdate: Invalid vector clock value got %d expected %d", interval, 1)
	}
}

func TestMergeVectorClock(t *testing.T) {
	vectorClock1 := NewVectorClock(3)
	vectorClock2 := NewVectorClock(3)

	vectorClock1.Update(0, 10)
	vectorClock1.Update(1, 3)
	vectorClock1.Update(2, 7)

	vectorClock2.Update(0, 10)
	vectorClock2.Update(1, 4)
	vectorClock2.Update(2, 6)

	vectorClock1.Merge(vectorClock2)

	intervalA, _ := vectorClock1.GetInterval(0)
	intervalB, _ := vectorClock1.GetInterval(1)
	intervalC, _ := vectorClock1.GetInterval(2)

	if intervalA != 10 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", intervalA, 10)
	}

	if intervalB != 4 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", intervalB, 4)
	}

	if intervalC != 7 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", intervalC, 7)
	}
}

func TestCopyVectorClock(t *testing.T) {
	vectorClock1 := NewVectorClock(3)

	vectorClock1.Update(0, 10)
	vectorClock1.Update(1, 3)
	vectorClock1.Update(2, 7)

	vectorClock2 := vectorClock1.Copy()

	vectorClock1.Update(0, 11)

	intervalA1, _ := vectorClock1.GetInterval(0)
	intervalA2, _ := vectorClock2.GetInterval(0)
	intervalB, _ := vectorClock2.GetInterval(1)
	intervalC, _ := vectorClock2.GetInterval(2)

	if intervalA1 != 11 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", intervalA1, 11)
	}

	if intervalA2 != 10 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", intervalA2, 10)
	}

	if intervalB != 3 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", intervalB, 3)
	}

	if intervalC != 7 {
		t.Errorf("[TEST] Invalid vector clock value got %d expected %d", intervalC, 7)
	}
}

func TestCompareVectorClocks(t *testing.T) {
	vectorClock1 := NewVectorClock(3)
	vectorClock2 := NewVectorClock(3)

	vectorClock1.Update(0, 10)
	vectorClock1.Update(1, 3)
	vectorClock1.Update(2, 7)

	vectorClock2.Update(0, 10)
	vectorClock2.Update(1, 2)
	vectorClock2.Update(2, 8)

	compareArr := vectorClock1.Compare(vectorClock2)

	if compareArr[0] != 0 {
		t.Errorf("[TEST] Invalid contents of IDs")
	}

	if compareArr[1] != 1 {
		t.Errorf("[TEST] Invalid contents of IDs")
	}

	if compareArr[2] != -1 {
		t.Errorf("[TEST] Invalid contents of IDs")
	}
}

func TestIsIdenticalVectorClocks(t *testing.T) {
	vectorClock1 := NewVectorClock(3)
	vectorClock2 := NewVectorClock(3)

	vectorClock1.Update(0, 10)
	vectorClock1.Update(1, 3)
	vectorClock1.Update(2, 7)

	vectorClock2.Update(0, 10)
	vectorClock2.Update(1, 2)
	vectorClock2.Update(2, 8)

	if vectorClock1.IsIdentical(vectorClock2) {
		t.Errorf("[TEST] Vector clocks identical when they are not supposed to be")
	}

	vectorClock2.Update(0, 10)
	vectorClock2.Update(1, 3)
	vectorClock2.Update(2, 7)

	if !vectorClock1.IsIdentical(vectorClock2) {
		t.Errorf("[TEST] Vector clocks not identical when they are supposed to be")
	}
}

func TestCovers(t *testing.T) {
	vc1 := ArrayToVC([]uint{1, 2, 3})
	vc2 := ArrayToVC([]uint{0, 1, 2})
	vc3 := ArrayToVC([]uint{0, 2, 2})

	vc1Cvc2 := vc1.Covers(vc2)
	vc2Cvc1 := vc2.Covers(vc1)
	vc1Cvc3 := vc1.Covers(vc3)

	if !vc1Cvc2 {
		t.Errorf("[TEST] Vector clocks covers. Expected %v to cover %v, ", vc1, vc2)
	}

	if vc2Cvc1 {
		t.Errorf("[TEST] Vector clocks covers. Expected %v not to cover %v, ", vc2, vc1)
	}

	if !vc1Cvc3 {
		t.Errorf("[TEST] Vector clocks covers. Expected %v to cover %v, ", vc1, vc3)
	}
}

func TestMin(t *testing.T) {
	vc1 := ArrayToVC([]uint{1, 2, 1})
	vc2 := ArrayToVC([]uint{0, 1, 2})
	vcmin := ArrayToVC([]uint{0, 1, 1})

	vc := vc1.Min(vc2)

	if !vcmin.IsIdentical(vc) {
		t.Errorf("[TEST] Vector clock Min. Expected %v for Min(%v,%v). Got %v, ", vcmin, vc1, vc2, vc)
	}
}
