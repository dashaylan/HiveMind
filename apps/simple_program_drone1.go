package main

import (
	"fmt"

	"../hivemind"
)

const (
	PID      = 0
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

	select {}
}
