/*
* Go implementation of the Pi approximation using integral.
* Adapted from https://pcj.icm.edu.pl/pi-approximation-using-integral
* for Hivemind DSM. Simple example to illustrate the use of locks.
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
	"os"
	"./hivemind"
	"strconv"
)

/*
#include "Tmk.h"
extern char *optarg;
int arrayDim = 100;
int *array;
*/
const (
	NRPROC   = 4
	NRBARR   = 5
	NRLOCKS  = 10
	NRPAGES  = 10
	PAGESIZE = 32
	PORT     = 2000
)

var done chan int = make(chan int, 2)
var gvec string = "Pi"

func f(x float64) float64 {
	return (4.0 / (1.0 + x*x))
}

func drone(id uint8, ids []uint8, ips []string) {

	var n, i, begin, end int
	var PI25DT float64 = 3.141592653589793238462643
	var mypi, w, sum, x float64
	var pi float64

	// Call startup function to create initialize the DSM and launch the remote processes
	nrProc := len(ids)
	hm := hivemind.NewHiveMind(id, uint8(nrProc), NRBARR, NRLOCKS, NRPAGES, PAGESIZE, "tipc")
	hm.SetDebug(4)

	hm.StartupTipc(PORT, gvec)

	time.Sleep(time.Millisecond * 1000)
	hm.ConnectToPeers(ids, ips)

	// Allocate shared memory block. We just need 1 float64 for this app
	size := int(unsafe.Sizeof(float64(0)))
	base, _ := hm.Malloc(size)

	// Number of iterations
	n = 50000000
	hm.Barrier(0)

	if id == 0 {
		pi = 0.0
		hm.WriteFloat(base, pi)
	}

	hm.Barrier(0)

	w = 1.0 / float64(n)
	sum = 0.0
	begin = n/nrProc*int(id) + 1
	end = n / nrProc * int(id+1)

	for i = begin; i <= end; i++ {
		x = w * (float64(i) - 0.5)
		sum += f(x)
	}
	mypi = w * sum

	// Get the lock to update the shared value of pi
	hm.LockAcquire(1)
	pi, _ = hm.ReadFloat(base)
	pi = mypi + pi
	hm.WriteFloat(base, pi)
	hm.LockRelease(1)
	hm.Barrier(0)

	// Print out result on first drone
	if id == 0 {
		pi, _ := hm.ReadFloat(base)
		fmt.Printf("pi is approximately %.16f, Error is %.16f\n", pi, math.Abs(pi-PI25DT))
	}

	hm.Exit()
	done <- int(id)
}

func main() {

	ips := make([]string, 16)
	/*
	for i := range ips {
		ips[i] = "localhost"
	}
	*/

	args := os.Args[1:]

	numIds, err := strconv.Atoi(args[1])

	usedIps := make([]string, numIds)


	if err != nil {
		fmt.Println(err)
	}

	ips = []string{
		"20.36.20.142",  //D1
		"20.190.48.49",  //D2
		"52.151.56.206", //D3
		"52.229.50.134", //D4
		"52.229.58.24",  //D5
		"40.65.105.92",  //D6
		"40.65.111.58",  //D7
		"52.151.41.54",  //D8
	}

	ids := make([]uint8, numIds)

	for i := range ids {
		ids[i] = uint8(i)
		usedIps[i] = ips[i]
	}






	go hivemind.DumpLog()
	//for _, id := range ids {
	i, err := strconv.Atoi(args[0])

	if err !=  nil{
		fmt.Println(err)
	}

	fmt.Println(ids, usedIps)
	go drone(uint8(i), ids, usedIps)
	//}


	id := <-done
	fmt.Printf("Drone[%d] is done\n", id)

	time.Sleep(time.Millisecond * 2000)

}
