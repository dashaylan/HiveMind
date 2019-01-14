/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the unit tests for WriteNoticeRecord
*/
package hivemind

import (
	"fmt"
	"testing"
)

////////////////////////////////////////////////////////////////////////////////////////////
// <DIFF TESTS>

func TestGetDiff(t *testing.T) {
	b1 := []byte{0xA1, 0xB2, 0xC3}
	b2 := []byte{0xA0, 0xB2, 0xC4}

	diff := GetDiff(b1, b2)

	if diff[0].Offset != 0 {
		t.Errorf("[TEST] Invalid Diff offset got %d expected %d", diff[0].Offset, 0)
	}

	if diff[0].Char != 0xA0 {
		t.Errorf("[TEST] Invalid Diff char got %x expected %x", diff[0].Char, 0xA0)
	}

	if diff[1].Offset != 2 {
		t.Errorf("[TEST] Invalid Diff offset got %d expected %d", diff[1].Offset, 2)
	}

	if diff[1].Char != 0xC4 {
		t.Errorf("[TEST] Invalid Diff char got %x expected %x", diff[1].Char, 0xC4)
	}
}

func TestApplyDiff(t *testing.T) {
	b1 := []byte{0xA1, 0xB2, 0xC3}
	b2 := []byte{0xA0, 0xB2, 0xC4}

	diff := GetDiff(b1, b2)

	ApplyDiff(b1, diff)

	if b1[0] != 0xA0 {
		t.Errorf("[TEST] Invalid byte in page got %x expected %x", b1[0], 0xA0)
	}

	if b1[1] != 0xB2 {
		t.Errorf("[TEST] Invalid byte in page got %x expected %x", b1[1], 0xB2)
	}

	if b1[2] != 0xC4 {
		t.Errorf("[TEST] Invalid byte in page got %x expected %x", b1[2], 0xC4)
	}
}

func TestInits(t *testing.T) {
	page1 := []byte("This a TESt senntnec")
	page2 := []byte("This a test Sentence")
	page3 := []byte("THIS A TEST SENTENCE")

	diff := GetDiff(page1, page2)
	diff2 := GetDiff(page2, page3)

	ApplyDiff(page1, diff)

	vc := NewVectorClock(4)
	interval := NewIntervalRecord(0, vc)

	wn1 := NewWriteNotice(0, interval)

	wn1.SetDiff(&diff)

	ApplyDiff(page1, diff2)

	fmt.Println(wn1)

	vc.Increment(0)

	interval2 := NewIntervalRecord(0, vc)

	wn2 := NewWriteNotice(0, interval2)

	wn2.SetDiff(&diff2)

	fmt.Println(wn2)

}

// </DIFF TESTS>
////////////////////////////////////////////////////////////////////////////////////////////
