package main

import (
	"fmt"

	"github.com/dashaylan/HiveMind/hivemind"
)

const (
	PID      = 99
	NRPROC   = 2
	NRBARR   = 1
	NRLOCKS  = 10
	NRPAGES  = 5
	PAGESIZE = 30
)

func main() {
	fmt.Println("!!! Simple Program !!!")
	go hivemind.DumpLog()

	fmt.Println("!!! Initializing HiveMind !!!")
	hm := hivemind.NewHiveMind(PID, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, "ipc")
	hm.SetDebug(4)
	fmt.Println("!!! HiveMind Initialized !!!")

	fmt.Println("!!! HiveMind Startup !!!")
	hm.Startup()
	fmt.Println("!!! HiveMind Startup Completed !!!")

	hm.Barrier(0)

	fmt.Println("!!! First barrier done !!!")

	if hm.GetProcID() == 1 {
		arrayDim := 32
		id := 1
		var start, end, i, array int

		array, _ = hm.Malloc(arrayDim)

		//hm.Barrier(0)
		start = (int(id) * (arrayDim / int(NRPROC)))
		end = int(id+1) * (arrayDim / int(NRPROC))

		if end > arrayDim {
			end = arrayDim
		}

		for i = start; i < end; i++ {
			hm.Write(array+i, uint8(i))
		}

		hm.Barrier(0)
	}

	select {}
}
