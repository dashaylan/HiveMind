package main

import (
	"fmt"

	"../hivemind"
)

const (
	PID      = 1
	NRPROC   = 3
	NRBARR   = 1
	NRLOCKS  = 10
	NRPAGES  = 5
	PAGESIZE = 30
)

var PROCADDR = []string{"111.1", "111.2", "111.3"}

func main() {
	fmt.Println("!!! Simple Program !!!")

	fmt.Println("!!! Initializing HiveMind !!!")
	hm := hivemind.NewHiveMind(PID, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, PROCADDR)
	fmt.Println("!!! HiveMind Initialized !!!")

	fmt.Println("!!! HiveMind Startup !!!")
	hm.Startup()
	fmt.Println("!!! HiveMind Startup Completed !!!")

	arrayDim := 32
	id := 1
	var start, end, i, array int

	array, _ = hm.Malloc(arrayDim)

	hm.Barrier(0)
	start = (int(id) * (arrayDim / int(NRPROC)))
	end = int(id+1) * (arrayDim / int(NRPROC))

	if end > arrayDim {
		end = arrayDim
	}

	for i = start; i < end; i++ {
		hm.Write(array+i, uint8(i))
	}

	hm.Barrier(0)

	fmt.Println("!!! Sent !!!")

	select {}
}
