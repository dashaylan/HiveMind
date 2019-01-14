/*
* Go implementation of the Pi approximation using integral.
* Adapted from https://pcj.icm.edu.pl/pi-approximation-using-integral
* for Hivemind DSM. Simple example to illustrate improvements to
* the performance that HiveMind can bring.
*
* The value of π is calculated using rectangles method that approximates
* following integral:
* π = 0 ∫ 1 4.0 / (1 + x2 ) dx
*
* In our code, the interval is divided into equal subintervals and we
* take top middle point of each subinterval to calculate area of the
* rectangle.
*
 */

package main

import (
	"fmt"
	"math"
	"time"
	"unsafe"

	"../hivemind"
)

const (
	NRPROC   = 8
	NRBARR   = 5
	NRLOCKS  = 10
	NRPAGES  = 10
	PAGESIZE = 32
)

var gvec string = "Pi"

func f(x float64) float64 {
	return (4.0 / (1.0 + x*x))
}

func main() {
	fmt.Println("!!! Hivemind Pi !!!")


	go hivemind.DumpLog()

	// Initialize drones (local and remote)
	hm := hivemind.NewHiveMind(0, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, "ipc")

	fmt.Println("!!! Done initializing !!!")
	hm.SetDebug(3)

	id := hm.GetProcID()
	nrproc := int(hm.GetNProcs())
	fmt.Println("!!!", nrproc, "drone(s) !!!")
	fmt.Println("!!! I am Drone", id, "!!!")

	// Start up
	hm.Startup(gvec)

	fmt.Println("!!! Started up !!!")

	// Prep for work
	var n, i, begin, end int
	var PI25DT float64 = 3.141592653589793238462643
	var mypi, w, sum, x float64
	var pi float64

	size := int(unsafe.Sizeof(float64(0)))
	base, _ := hm.Malloc(size)

	n = 50000000 // Number of iterations

	fmt.Println("!!!", n, "iterations !!!")

	fmt.Println("!!! Preparation completed !!!")

	hm.Barrier(0) // 1st barrier

	start := time.Now()

	fmt.Println("!!! Finished 1st barrier !!!")

	// First drone's work
	if id == 0 {
		pi = 0.0
		hm.WriteFloat(base, pi)
		fmt.Println("!!! Finished drone 1 only preparation !!!")
	}

	hm.Barrier(0) // 2nd barrier

	fmt.Println("!!! Finished 2nd barrier !!!")

	fmt.Println("!!! Start working !!!")

	// Work
	w = 1.0 / float64(n)
	sum = 0.0
	begin = n/nrproc*int(id) + 1
	end = n / nrproc * int(id+1)

	for i = begin; i <= end; i++ {
		x = w * (float64(i) - 0.5)
		sum += f(x)
	}
	mypi = w * sum

	fmt.Println("!!! Done working, aiming to sync !!!")

	// Get the lock to update the shared value of pi
	hm.LockAcquire(1)
	pi, _ = hm.ReadFloat(base)
	pi = mypi + pi
	hm.WriteFloat(base, pi)
	hm.LockRelease(1)
	hm.Barrier(0) // 3rd barrier

	fmt.Println("!!! Finished 3rd barrier !!!")

	if id == 0 {
		pi, _ := hm.ReadFloat(base)
		fmt.Printf("pi is approximately %.16f, Error is %.16f\n", pi, math.Abs(pi-PI25DT))
		fmt.Println("Time elapsed:", time.Now().Sub(start), "with", nrproc, "drone(s).")
	}

	hm.Barrier(0)

	fmt.Println("!!! All done !!!")

	hm.Exit()
}
