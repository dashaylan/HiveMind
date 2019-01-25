/*
Package main which implements a quicksort application using the HiveMind DSM

*/
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/dashaylan/HiveMind/hivemind"
)

type WorkLoad struct {
	Start int
	End   int
	Id    uint8
	Base  int
}

const (
	NRBARR         = 5
	NRLOCKS        = 10
	NRPAGES        = 10
	PAGESIZE       = 32
	PORT           = 2000
	WORKADDROFFSET = 164
	LENGTH         = 25
	WORKQLENGTH    = 128
)

var a []int32

var done chan int = make(chan int, 2)

func initDrone(id uint8, ids []uint8, ips []string) {
	nrProc := len(ids)
	hm := hivemind.NewHiveMind(id, uint8(nrProc), NRBARR, NRLOCKS, NRPAGES, PAGESIZE, ips)
	hm.StartupTipc(PORT, "") // anything other than an empty string produces an EOF error

	time.Sleep(time.Millisecond * 1000)
	hm.ConnectToPeers(ids, ips)

	piv := LENGTH - 1
	piv = compareSwap(a, 0, piv)

	size := int(unsafe.Sizeof(int32(0))) * LENGTH
	base, _ := hm.Malloc(size)

	hm.LockAcquire(0)
	hm.WriteN(base, arrToBytes(a))

	wq := make([]byte, WORKQLENGTH)
	queue(0, piv-1, wq)
	queue(piv+1, LENGTH-1, wq)
	hm.WriteN(WORKADDROFFSET, wq)
	hm.LockRelease(0)

	hm.Barrier(1)

	hm.Barrier(0)
	hm.LockAcquire(0)
	buf, _ := hm.ReadN(base, LENGTH*int(unsafe.Sizeof(int32(0))))
	hm.LockRelease(0)
	arr := bytesToArr(buf)
	fmt.Println("Sorted: ", arr)
	done <- int(id)
}

func drone(id uint8, ids []uint8, ips []string) {
	nrProc := len(ids)
	hm := hivemind.NewHiveMind(id, uint8(nrProc), NRBARR, NRLOCKS, NRPAGES, PAGESIZE, ips)
	hm.StartupTipc(PORT, "")

	time.Sleep(time.Millisecond * 1000)
	hm.ConnectToPeers(ids, ips)

	hm.Barrier(1)
	hm.LockAcquire(0)
	wq, _ := hm.ReadN(WORKADDROFFSET, WORKQLENGTH)
	start, end := dequeue(wq)
	hm.WriteN(WORKADDROFFSET, wq)
	if start == -1 && end == -1 {
		hm.LockRelease(0)
		hm.Barrier(0)
		done <- int(id)
		return
	}

	work(start, end, hm)
	hm.Barrier(0)
	done <- int(id)
}

func work(start, end int, hm *hivemind.HM) {
	buf, _ := hm.ReadN(0, LENGTH*int(unsafe.Sizeof(int32(0))))

	arr := bytesToArr(buf)

	if end < 0 || start >= len(arr) || start > end {
		hm.LockRelease(0)

		return
	}

	if end-start < 2 {
		if arr[start] > arr[end] {
			startVal := arr[start]
			arr[start] = arr[end]
			arr[end] = startVal
			hm.WriteN(0, arrToBytes(arr))
		}

		hm.LockRelease(0)
		return
	}

	piv := compareSwap(arr, start, end)

	hm.WriteN(0, arrToBytes(arr))

	wq, _ := hm.ReadN(WORKADDROFFSET, WORKQLENGTH)
	queue(start, piv-1, wq)
	queue(piv+1, end, wq)
	hm.WriteN(WORKADDROFFSET, wq)
	hm.LockRelease(0)
}

func main() {
	ids := []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	ips := []string{"localhost", "localhost", "localhost", "localhost", "localhost", "localhost", "localhost",
		"localhost", "localhost", "localhost", "localhost", "localhost", "localhost", "localhost",
		"localhost", "localhost", "localhost", "localhost", "localhost", "localhost", "localhost"}

	a = make([]int32, LENGTH)

	for i := 0; i < LENGTH; i++ {
		a[i] = rand.Int31n(100)
	}
	fmt.Println("Inital: ", a)
	go hivemind.DumpLog()
	go initDrone(0, ids, ips)
	for _, id := range ids {
		if id == 0 {
			continue
		}
		go drone(id, ids, ips)
	}

	for i := 0; i < len(ids); i++ {
		id := <-done
		fmt.Printf("Drone[%d] is done\n", id)
	}
	time.Sleep(time.Millisecond * 2000)
}

func compareSwap(arr []int32, start, piv int) int {
	ind := start
	pivVal := arr[piv]
	for i := start; i < piv; i++ {

		if arr[i] <= pivVal {
			indVal := arr[ind]
			arr[ind] = arr[i]
			arr[i] = indVal
			ind++
		}
	}

	arr[piv] = arr[ind]
	arr[ind] = pivVal

	return ind
}

func arrToBytes(a []int32) []byte {
	buf := new(bytes.Buffer)
	for _, d := range a {
		binary.Write(buf, binary.LittleEndian, d)
	}
	return buf.Bytes()
}

func bytesToArr(b []byte) (a []int32) {
	r := bytes.NewReader(b)
	a = make([]int32, len(b)/4)
	binary.Read(r, binary.LittleEndian, &a)

	return a
}

func posToBytes(start, end int) []byte {
	buf := new(bytes.Buffer)
	arr := []int{start, end}
	for _, v := range arr {
		binary.Write(buf, binary.LittleEndian, int32(v))
	}

	return buf.Bytes()
}

func bytesToPos(b []byte) (int, int) {
	r := bytes.NewReader(b)
	a := make([]int32, 2)
	binary.Read(r, binary.LittleEndian, &a)
	return int(a[0]), int(a[1])
}

func dequeue(b []byte) (int, int) {
	s, e := bytesToPos(b[0:8])

	if s == 0 && e == 0 {
		return -1, -1
	}

	for i := 0; i < len(b); i += 8 {
		for k := i; k < i+8; k++ {
			b[k] = b[k+8]
		}

		if allZero(b[i+8 : i+16]) {
			break
		}

	}

	return s, e
}

func queue(start, end int, b []byte) {
	v := posToBytes(start, end)

	var i int
	for i = 0; i < len(b); i += 8 {
		if allZero(b[i : i+8]) {
			break
		}
	}

	for k := i; k < i+8; k++ {
		b[k] = v[k-i]
	}

}

func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}
