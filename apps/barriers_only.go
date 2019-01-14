package main

import (
	"fmt"

	"../hivemind"
)

const (
	PID      = 99
	NRPROC   = 8
	NRBARR   = 1
	NRLOCKS  = 10
	NRPAGES  = 5
	PAGESIZE = 30
)

func main() {
	fmt.Println("!!! Barriers only !!!")
	go hivemind.DumpLog()

	fmt.Println("!!! Initializing HiveMind !!!")
	hm := hivemind.NewHiveMind(PID, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, "ipc")

	fmt.Println("!!! HiveMind Debug level set !!!")
	hm.SetDebug(4)

	fmt.Println("!!! HiveMind Startup !!!")
	hm.Startup()

	fmt.Println("!!! First barrier !!!")
	hm.Barrier(0)

	fmt.Println("!!! Second barrier !!!")
	hm.Barrier(0)

	fmt.Println("!!! Third barrier !!!")
	hm.Barrier(0)

	select {}
}
