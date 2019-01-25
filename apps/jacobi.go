/*
Package main which implements Jacobi iteration -- one of the benchmark programs
used to test TreadMarks. This is the Go version of the C implementation shown in
the paper by Keleher et al. The Jacobi Iteration illustrates the use of barriers.

Jacobi is a method for solving partial differential equations. The example iterates
over a two dimensional array. During each iteration, every matrix element is updated
to the average of its nearest neighbours (above, below, left and right). Jacobi uses
a scratch array to store the new values computed during each iteration, so as to avoid
overwriting the old value of the element before it is used by the neighbor. In the
parallel version, all processors are assigned roughly equal sized bands of rows.
The rows on the boundary of a band are shared by two neighbouring processes.

The TreadMarks version uses two arrays: a grid and a scratch array. Grid is allocated
in shared memory, while scratch is private to each process. grid is allocated and
initialized by process 0. Synchronization in Jacobi is done by means of barriers.


*/

package main

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/dashaylan/HiveMind/hivemind"
)

// M is the number of rows in the grid
const M = 16

// N is the number of columns in the grid
const N = 16

// NIterations is the number of iterations to perform
const NIterations = 3

var scratch [M][N]float64

var done chan int = make(chan int, 2)

const (
	NRBARR   = 2
	NRLOCKS  = 10
	NRPAGES  = (M * N)
	PAGESIZE = 128
	PORT     = 2000
)

func offset(i, j int) int {
	return (i*N + j) * int(unsafe.Sizeof(float64(0)))
}

var gvec string = "Jacobi"

func drone(id uint8, ids []uint8, ips []string) {

	// Call startup function to create initialize the DSM and launch the remote processes
	nrProc := len(ids)
	hm := hivemind.NewHiveMind(id, uint8(nrProc), NRBARR, NRLOCKS, NRPAGES, PAGESIZE, "tipc")
	hm.StartupTipc(PORT, gvec)

	time.Sleep(time.Millisecond * 1000)
	hm.ConnectToPeers(ids, ips)

	// Allocate shared memory block
	size := (M * N) * int(unsafe.Sizeof(float64(0)))
	base, _ := hm.Malloc(size)

	length := M / nrProc
	begin := length * int(id)
	end := length * int(id+1)

	fmt.Println("Mem", id, size, base, length, begin, end)
	hm.Barrier(0)

	for iter := 0; iter < NIterations; iter++ {

		// We need to guard against going outside of the
		// boundary of the matrix.
		for i := begin + 1; i < end-1; i++ {
			var divisor float64 = 4
			var a, b, c, d float64
			for j := 1; j < N-1; j++ {
				a, _ = hm.ReadFloat(offset(i-1, j))
				b, _ = hm.ReadFloat(offset(i+1, j))
				c, _ = hm.ReadFloat(offset(i, j-1))
				d, _ = hm.ReadFloat(offset(i, j+1))
				scratch[i][j] = (a + b + c + d) / divisor
			}
		}
		hm.Barrier(1)

		for i := begin; i < end; i++ {
			for j := 0; j < N; j++ {
				hm.WriteFloat(offset(i, j), scratch[i][j])
			}
		}

		hm.Barrier(2)
	}
	hm.Exit()
	done <- int(id)
}

func main() {
	ids := []uint8{0, 1}
	ips := []string{"localhost", "localhost"}

	go hivemind.DumpLog()
	go drone(0, ids, ips)
	go drone(1, ids, ips)

	for i := 0; i < len(ids); i++ {
		id := <-done
		fmt.Printf("Drone[%d] is done\n", id)
	}
	time.Sleep(time.Millisecond * 2000)

}
