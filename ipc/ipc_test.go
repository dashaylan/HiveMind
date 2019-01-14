/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the unit tests for the inter-processor
messaging layer.
*/
package ipc

import (
	"fmt"
	"testing"

	"../configs"
)

func TestRunAll(t *testing.T) {
	TestCopyStartSingleDrone(t)
	TestCopyStartMultiDrone(t)
}

func TestStartCBMServer(t *testing.T) {
	_, cbmStartError := StartCBMServer()
	if cbmStartError != nil {
		t.Errorf("[TEST] Can't not start barrier server, got error: ", cbmStartError.Error())
	} else {
		fmt.Println("[TEST] Barrier server started")
	}
}

func TestCopyStartSingleDrone(t *testing.T) {
	//move up directory
	//normal app would be called from the directory above
	runComm("cd", []string{".."}, false)
	testList := []configs.DroneManagerConfig{configs.DroneManagerConfig{Username: "drone1", Password: "DroneDrone01", Address: "20.36.20.142"}}
	num, _, err := StartNodes(testList)
	if err != nil {
		t.Errorf("[TEST] Unable to deploy to", testList[0].Address, ", got error", err.Error())
	}
	fmt.Println("[TEST] Deployed on", num, "machines, out of", len(testList), "machines")
}

func TestCopyStartMultiDrone(t *testing.T) {
	//move up directory
	//normal app would be called from the directory above
	runComm("cd", []string{".."}, false)
	testList := []configs.DroneManagerConfig{configs.DroneManagerConfig{Username: "drone1", Password: "DroneDrone01", Address: "20.36.20.142"},
		configs.DroneManagerConfig{Username: "drone2", Password: "DroneDrone02", Address: "20.190.48.49"}}
	num, _, err := StartNodes(testList)
	if err != nil {
		t.Errorf("[TEST] Unable to deploy to", testList[0].Address, ", got error", err.Error())
	}
	fmt.Println("[TEST] Deployed on", num, "machines, out of", len(testList), "machines")
}
