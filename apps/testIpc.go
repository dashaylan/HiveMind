package main

import (
	"github.com/dashaylan/HiveMind/hivemind"
)

const (
	NRPROC   = 2
	NRBARR   = 2
	NRLOCKS  = 10
	NRPAGES  = 10
	PAGESIZE = 32
	PORT     = 2000
)

func main() {
	hm := hivemind.NewHiveMind(0, NRPROC, NRBARR, NRLOCKS, NRPAGES, PAGESIZE, "")
	hm.Startup()

	select {}
}
