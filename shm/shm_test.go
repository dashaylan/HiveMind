/*
Package shm implements shared virtual memory manager.

This file implements the unit tests for the Shared virtual memory manager

*/
package shm

import (
	"testing"
	//"fmt"
)

func TestSizes(t *testing.T) {
	P, N := 1024, 64
	S := P * N
	vm := NewShm(P, N)
	size := vm.GetSize()
	psize := vm.GetPageSize()
	if size != S {
		t.Errorf("Mem size was incorrect, got: %d, want: %d.", size, S)
	}
	if psize != P {
		t.Errorf("Page size was incorrect, got: %d, want: %d.", psize, P)
	}
}
func TestMallocFree(t *testing.T) {
	P, N := 1024, 64
	vm := NewShm(P, N)
	size := P * N
	ptr := []int{512, 1024, 2048, 65, 1005, 110, 130, 321, 2200}
	addr := 0

	// Allocate a list of block
	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	// Free blocks allocated above
	freePtr := 0
	for _, p := range ptr {
		e1 := vm.Free(freePtr)
		//vm.DumpFreeList()
		//vm.DumpAllocList()
		if size > p {
			if e1 != nil {
				t.Errorf("Free returned error for %d, got: %s, want: <nil>", freePtr, e1)
			}
			freePtr += p
		} else {
			if e1 == nil {
				t.Errorf("Free did not return error for invalid address %d, got: %s, want: error.", p+freePtr, e1)
			}
		}
	}
}

func TestMallocPositiveOoB(t *testing.T) {
	P, N := 1024, 64
	vm := NewShm(P, N)
	size := P * N

	//Attempt to malloc outside of positive Shm range
	ptr := []int{size + 1, size * 2, size + 3, size * size}
	for _, p := range ptr {
		m, e := vm.Malloc(p)
		if e == nil {
			t.Errorf("[Postive OoB Malloc] Malloc did not return error for invalid memory size %d, got %s, want: error.",
				m, e)
		}
	}
}

func TestMallocNegativeOoB(t *testing.T) {
	P, N := 1024, 64
	vm := NewShm(P, N)
	size := P * N

	//Attempt to malloc outside of negative Shm range
	ptr := []int{-(size + 1), -(size * 2), -(size * size)}
	for _, p := range ptr {
		m, e := vm.Malloc(p)
		//vm.DumpFreeList()
		//vm.DumpAllocList()
		if e == nil {
			t.Errorf("[Negative OoB Malloc] Malloc did not return error for invalid memory size %d, got %s, want: error.",
				m, e)
		}
	}
}

func TestFreeStart(t *testing.T) {
	P, N := 1024, 64
	vm := NewShm(P, N)
	//size := P * N

	//Attempt to free from non-malloc'd chunk
	ptr := []int{10, 20, 50, 100}

	freePtr := 0
	for _, p := range ptr {
		e1 := vm.Free(p)
		freePtr += p
		if e1 == nil {
			t.Errorf("Free did not return error for invalid address %d, got: %s, want: error.", p+freePtr, e1)
		}
	}

}

func TestFreeNegative(t *testing.T) {
	P, N := 1024, 64
	vm := NewShm(P, N)
	//size := P * N

	//Attempt to free from non-malloc'd chunk
	ptr := []int{-10, -20, -50, -100}

	freePtr := 0
	for _, p := range ptr {
		e1 := vm.Free(p)
		freePtr += p
		if e1 == nil {
			t.Errorf("Free did not return error for invalid address %d, got: %s, want: error.", p+freePtr, e1)
		}
	}

}

func TestFreeDoubleAndOoB(t *testing.T) {
	P, N := 1024, 64
	vm := NewShm(P, N)
	size := P * N
	addr := 0

	//Attempt to free from non-malloc'd chunk
	ptr := []int{140, 3320, 250}

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	ptr2 := []int{3320, 152352600}

	freePtr := 0
	for _, p := range ptr {
		e1 := vm.Free(freePtr)
		//vm.DumpFreeList()
		//vm.DumpAllocList()
		if size > p {
			if e1 != nil {
				t.Errorf("Free returned error for %d, got: %s, want: <nil>", freePtr, e1)
			}
			freePtr += p
		} else {
			if e1 == nil {
				t.Errorf("Free did not return error for invalid address %d, got: %s, want: error.", p+freePtr, e1)
			}
		}
	}

	for _, p := range ptr2 {
		e1 := vm.Free(freePtr)
		if e1 == nil {
			t.Errorf("Free did not return error for invalid address %d, got: %s, want: error.", p+freePtr, e1)
		}
	}
}

type TestVM struct {
	M *Shm
}

func (t *TestVM) SEGVHandlerTest(addr int, length int, fault Prot) error {
	return t.M.MProtect(addr, length, fault)
}

func TestReadWrite0(t *testing.T) {
	P, N := 1024, 64
	tstVM := TestVM{NewShm(P, N)}
	vm := tstVM.M
	size := P * N
	addr := 0
	byteEx := byte('A')

	ptr := []int{100, 600, 400, 435, 2400}
	ptrstart := make([]int, 0)

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			ptrstart = append(ptrstart, p1)
			vm.InstallSEGVHandler(tstVM.SEGVHandlerTest)

			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	for _, start := range ptrstart {
		e := vm.Write(start, byteEx)
		if e != nil {
			t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
		}
	}
}

func TestWrite(t *testing.T) {
	P, N := 30, 5
	tstVM := TestVM{NewShm(P, N)}
	vm := tstVM.M
	size := P * N
	addr := 0
	byteEx := byte('A')
	byteEx2 := byte('X')

	ptr := []int{5, 5, 5, 10, 10, 10}
	ptrstart := make([]int, 0)

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			ptrstart = append(ptrstart, p1)
			vm.InstallSEGVHandler(tstVM.SEGVHandlerTest)

			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	for index, start := range ptrstart {
		if index%2 == 0 {
			e := vm.Write(start, byteEx2)
			if e != nil {
				t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
			}
		} else {
			e := vm.Write(start, byteEx)
			if e != nil {
				t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
			}

		}
	}

	//vm.DumpPages()
}

func TestWriteOoB(t *testing.T) {
	P, N := 30, 5
	tstVM := TestVM{NewShm(P, N)}
	vm := tstVM.M
	size := P * N
	addr := 0
	byteEx := byte('A')
	byteEx2 := byte('X')

	ptr := []int{20, 20, 30, 40, 10}
	oob := []int{size + 2, size * size, -size, -333}
	ptrstart := make([]int, 0)

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			ptrstart = append(ptrstart, p1)
			vm.InstallSEGVHandler(tstVM.SEGVHandlerTest)

			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	for _, start := range oob {
		e := vm.Write(start, byteEx2)
		if e == nil {
			t.Errorf("Expected OutofBounds Error for %d at %d.", byteEx, start)
		}
	}
}

func TestWriteN(t *testing.T) {
	P, N := 30, 5
	tstVM := TestVM{NewShm(P, N)}
	vm := tstVM.M
	size := P * N
	addr := 0
	byteEx := []byte("This is a test sentence")

	ptr := []int{30, 30}
	ptrstart := make([]int, 0)

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			ptrstart = append(ptrstart, p1)
			vm.InstallSEGVHandler(tstVM.SEGVHandlerTest)

			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	for _, start := range ptrstart {
		e := vm.WriteN(start, byteEx)
		if e != nil {
			t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
		}
	}

	//vm.DumpPages()

}

func TestWriteNPageOver(t *testing.T) {
	P, N := 30, 5
	tstVM := TestVM{NewShm(P, N)}
	vm := tstVM.M
	size := P * N
	addr := 0
	byteEx := []byte("This is a test sentence which is longer than the page size for this test")
	byteEx2 := []byte("Really long sentence number two to test overwriting")

	ptr := []int{30, 30}
	ptrstart := make([]int, 0)

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			ptrstart = append(ptrstart, p1)
			vm.InstallSEGVHandler(tstVM.SEGVHandlerTest)

			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	/*

		for _, start := range ptrstart {
			e := vm.WriteN(start, byteEx)
			if e != nil {
				t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
				}
			}
	*/

	vm.WriteN(ptrstart[0], byteEx)
	vm.WriteN(ptrstart[1], byteEx2)

	vm.DumpPages()

}

func TestRead(t *testing.T) {
	P, N := 30, 5
	tstVM := TestVM{NewShm(P, N)}
	vm := tstVM.M
	size := P * N
	addr := 0
	byteEx := byte('A')
	byteEx2 := byte('X')

	ptr := []int{20, 20, 30, 40, 10}
	ptrstart := make([]int, 0)

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			ptrstart = append(ptrstart, p1)
			vm.InstallSEGVHandler(tstVM.SEGVHandlerTest)

			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	for index, start := range ptrstart {
		if index%2 == 0 {
			e := vm.Write(start, byteEx2)
			if e != nil {
				t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
			}
		} else {
			e := vm.Write(start, byteEx)
			if e != nil {
				t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
			}

		}
	}

	for index, start := range ptrstart {
		if index%2 == 0 {
			b, e := vm.Read(start)
			if e != nil || b != byteEx2 {
				t.Errorf("Read returned an error: [%s]. \n Expected %d at %d found %d.", e, byteEx2, start, b)
			}
		} else {
			b, e := vm.Read(start)
			if e != nil || b != byteEx {
				t.Errorf("Read returned an error: [%s]. \n Expected %d at %d found %d.", e, byteEx, start, b)
			}

		}
	}
}

func TestReadOoB(t *testing.T) {
	P, N := 30, 5
	tstVM := TestVM{NewShm(P, N)}
	vm := tstVM.M
	size := P * N
	addr := 0
	byteEx := byte('A')
	byteEx2 := byte('X')

	ptr := []int{20, 20, 30, 40, 10}
	oob := []int{size + 2, size * size, -size, -333}
	ptrstart := make([]int, 0)

	for _, p := range ptr {
		p1, e1 := vm.Malloc(p)
		if size-addr > p {
			if p1 != addr || e1 != nil {
				t.Errorf("Malloc returned incorrect address, got: %d, want: %d.", p1, addr)
			}
			ptrstart = append(ptrstart, p1)
			vm.InstallSEGVHandler(tstVM.SEGVHandlerTest)

			addr += p
		} else {
			if p1 != -1 || e1 == nil {
				t.Errorf("Malloc returned valid address instead of error, got: %d, want: %d.", p1, -1)
			}
		}
	}

	for index, start := range ptrstart {
		if index%2 == 0 {
			e := vm.Write(start, byteEx2)
			if e != nil {
				t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
			}
		} else {
			e := vm.Write(start, byteEx)
			if e != nil {
				t.Errorf("Write returned an error: [%s] for valid write of %d at %d.", e, byteEx, start)
			}

		}
	}

	for _, start := range oob {
		_, e := vm.Read(start)
		if e == nil {
			t.Errorf("Expected OutofBounds Error for at %d.", start)
		}
	}
}

func TestMProtect(t *testing.T) {

}

func TestHandlers(t *testing.T) {

}

func TestPageReadWrite(t *testing.T) {

}
