/*
* Go implementation of the simple DSM app in Figure 2.1 in Keleher thesis
*
 */
package main

import (
	"fmt"
	"time"

	"../hivemind"
)

/*
#include "Tmk.h"
extern char *optarg;
int arrayDim = 100;
int *array;
*/
const (
	NRPROC   = 2
	NRBARR   = 5
	NRLOCKS  = 10
	NRPAGES  = 10
	PAGESIZE = 32
	PORT     = 2000
)

var gvec string = "SimpleDSM"
var done chan int = make(chan int, 2)

func drone(id uint8, ids []uint8, ips []string) {
	arrayDim := 80
	var start, end, i, array int
	hm := hivemind.NewHiveMind(id, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, "tipc")
	hm.SetDebug(3)
	hm.StartupTipc(PORT, gvec)

	time.Sleep(time.Millisecond * 1000)
	hm.ConnectToPeers(ids, ips)

	array, _ = hm.Malloc(arrayDim)
	//Tmk_distribute(&array, sizeof(array)); /* Send 4-byte ptr value */

	//Tmk_barrier(0);
	hm.Barrier(0)
	// start = Tmk_proc_id * (arrayDim / Tmk_nprocs);
	start = (int(id) * (arrayDim / int(NRPROC)))

	//end = (Tmk_proc_id + 1) * (arrayDim / Tmk_nprocs);
	end = int(id+1) * (arrayDim / int(NRPROC))
	//if (end > arrayDim) end = arrayDim;
	if end > arrayDim {
		end = arrayDim
	}

	//for (i = start; i < end; i++)
	//array[i] = i;
	for i = start; i < end; i++ {
		hm.Write(array+i, uint8(i))
	}

	//Tmk_barrier(0);
	hm.Barrier(0)
	//Tmk_exit(0);
	//data, _ := hm.ReadN(0, arrayDim)

	if id == 0 {
		buf, _ := hm.ReadN(array, arrayDim)
		fmt.Printf("Array = %v\n", buf)
	}

	hm.Exit()
	//fmt.Println("Array[", id, "]", data)
	done <- int(id)
}

func main() {
	ids := []uint8{0, 1}
	ips := []string{"localhost", "localhost"}

	go hivemind.DumpLog()
	go drone(0, ids, ips)
	go drone(1, ids, ips)

	for i := 0; i < NRPROC; i++ {
		id := <-done
		fmt.Printf("Drone[%d] is done\n", id)
	}
	time.Sleep(time.Millisecond * 2000)

}
