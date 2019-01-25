/*
Package hivemind implements the TreadMarks API.

This file contains structs and functions to manipulate configuration JSONs
*/
package configs

import (
	"encoding/json"
	"io/ioutil"
)

//for each drone we intend to deploy,
//we need these to connect to it
type DroneManagerConfig struct {
	Address  string
	Port     string
	Username string
	Password string
}

//these are the DroneList seen by other drones
type DroneConfig struct {
	Address string
	PID     uint8
}

//this is the struct for config.json
//there are two possibilities:
//we're the first drone, and hence assumes CBM role
//in this case, IsCBM == true, CBM is undefined, and Drones is the other machines we will deploy to
//seconde case we are a drone being deployed
//in this case, IsCBM == false, CBM is the machine deploying us, and DroneList is the list of drones to connect to
type Config struct {
	IsCBM     bool                 //Are we the first drone (ie. CBM) or just another drone?
	Drones    []DroneManagerConfig //either peers or machines to deploy on
	CBM       string               //address of the central barrier manager
	DroneList []DroneConfig        //list of drone addresses
}

//reads configuration from config.json
func ReadConfig() (Config, error) {
	c := Config{}
	cfFile, err := ioutil.ReadFile("config.json")
	if err != nil {
		//fail to read config
		return c, err
	}
	err = json.Unmarshal(cfFile, &c)
	if err != nil {
		//unable to decode the config
		return c, err
	}

	return c, nil
}

func ReadDroneConfig() (Config, error) {
	c := Config{}
	cfFile, err := ioutil.ReadFile("droneConf.json")
	if err != nil {
		//fail to read config
		return c, err
	}
	err = json.Unmarshal(cfFile, &c)
	if err != nil {
		//unable to decode the config
		return c, err
	}

	return c, nil
}

//typically the first drone has its config
//prepared beforehand
//this func is to write configs for drones
func WriteConfig(filename string, c Config) error {
	cfArr, err := json.Marshal(c)
	if err != nil {
		//failed to encode the config
		return err
	}
	err = ioutil.WriteFile(filename, cfArr, 0644)
	return err
}
