/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the inter-processor messaging
layer. This implements a fast, reliable point-to-point communications
between this node and all other nodes in the DSM
*/
package ipc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/dashaylan/HiveMind/configs"

	"golang.org/x/crypto/ssh"
)

/* IPC TODO - See Design.md for more details

The hivemind.go file has some pseudocode describing what
the functions over there need to do. Generally, we need
send and receieve functions that the drones can utilize.

The IPC module should be lower level communication while
HiveMind itself should be doing a lot of the logic that
is being done here. So we need to move some of this stuff
out. Additionally, the communication system can't rely
on rpc calls due to all of them blocking until a response
is sent. This will serialize everything and have a pretty
large impact on performance.

From a networking point of view, this is more like link
layer communication.


*/

// Ipc structure defines the properties of an Ipc instance
type Ipc struct {
	pid     uint8       // id of this drone
	rxChan  chan []byte // Rx channel
	txChan  chan []byte // Tx Channel
	peermap PeerMap     //Peermaps
	//....
}

//how many ipc messages to buffer
const ipcBufSize int = 64

//the port on which drones listens to others
const p2pport string = ":6464"

// Temporary all ints, please modify to the type you needed when use
type IpcArgs struct {
	VC       []uint
	Interval int
	PageID   int
	DiffID   int
	Diff     int
	PageCopy int
	Copyset  int
	// local Intervals lastLt & VC[pid] // Need to clarify on what it is.
}

type IpcMessage struct {
	MessageId int
	SourceId  int
	DestId    int
	Args      IpcArgs
}

type Peer struct {
	Conn    net.Conn
	Address string
	Pid     uint8
}

type PeerMap struct {
	mut      *sync.RWMutex
	internal map[uint8]Peer
}

// Initialize components of PeerMap
func (pm *PeerMap) Initialize() {
	pm.mut = new(sync.RWMutex)
	pm.internal = make(map[uint8]Peer)
}

// Add peer and its connection to map
func (pm *PeerMap) AddPeer(pid uint8, peer Peer) {
	pm.mut.Lock()
	pm.internal[pid] = peer
	pm.mut.Unlock()
}

// Return peer, and its existence
func (pm *PeerMap) GetPeer(pid uint8) (Peer, bool) {
	pm.mut.RLock()
	peer, exist := pm.internal[pid]
	pm.mut.RUnlock()
	return peer, exist
}

// Return conn of peer, and its existence
func (pm *PeerMap) GetPeerConn(pid uint8) (net.Conn, bool) {
	pm.mut.RLock()
	peer, exist := pm.internal[pid]
	pm.mut.RUnlock()
	if exist {
		return peer.Conn, exist
	} else {
		return nil, exist
	}
}

// Remove peer and its connection from map
func (pm *PeerMap) RemovePeer(pid uint8) {
	pm.mut.Lock()
	delete(pm.internal, pid)
	pm.mut.Unlock()
}

// Number of peers (might be handy)
func (pm *PeerMap) NumPeers() int {
	pm.mut.RLock()
	numPeers := len(pm.internal)
	pm.mut.RUnlock()
	return numPeers
}

//Return a copy of PeerMap
//Useful for iteration
func (pm *PeerMap) DeepCopy() PeerMap {
	pm.mut.RLock()
	newCopy := PeerMap{mut: new(sync.RWMutex), internal: pm.internal}
	pm.mut.RUnlock()
	return newCopy
}

//returns all the keys
//useful for iteration
func (pm *PeerMap) KeyList() []uint8 {
	pm.mut.RLock()
	arr := make([]uint8, len(pm.internal))
	for ind := range pm.internal {
		arr = append(arr, ind)
	}
	pm.mut.RUnlock()
	return arr
}

// Initialize PeerMap
func InitPeerMap() PeerMap {
	peermap := *new(PeerMap)
	peermap.Initialize()
	return peermap
}

//runs a native exec, possibly dumping output as required
func runComm(command string, arg []string, debug bool) error {
	com := exec.Command(command, arg...)
	fmt.Println("Runcomm: ", com)
	//dump output to console, for debugging

	var stderr bytes.Buffer

	com.Stderr = &stderr

	err := com.Run()
	if err != nil {
		log.Fatal(fmt.Sprint(err) + ": " + stderr.String())
	}
	return err
}

//https://gist.github.com/jniltinho/9788121
//function to get the public ip address
//we assume we are connected to external internet
func GetOutboundIP() string {
	resp, err := http.Get("http://ipv4.myexternalip.com/raw")
	if err != nil {
		fmt.Println("GetOutboundIP error,", err)
		return "[::]"
	}
	defer resp.Body.Close()
	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	// if err != nil {
	// 	fmt.Println("GetOutboundIP error 2,", err)
	// 	return "[::]"
	// }
	return string(buf[:n-1])
	//fmt.Println("GetOutboundIP is", resp.Body)
}

// GetOutboundIPKvm Function to grab outbound IP for use with KVM system
func GetOutboundIPKvm() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	fmt.Printf("Address %v\n", localAddr.IP.String())
	return localAddr.IP.String()
}

func remoteComm(connection *ssh.Client, command string) error {
	session, err := connection.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return err
	}

	return session.Run(command)
}

type droneConfInternal struct {
	ManConf configs.DroneManagerConfig
	PID     uint8
}

//given a list of IP address, start drones on each machine
//also requires username:password for ssh
//password as hardcoded string is bad for security,
//but acceptable for project purposes
//returns number of drones successfully started
//returns a mapping of drone addresses to pid
func StartNodes(inList []configs.DroneManagerConfig) (int, []configs.DroneConfig, error) {
	//construct the config file for drones
	//included: it is not the first drone, it has peers, it has CBM
	pidc := uint8(1)
	//we use this to correlate DroneManagerConfigs and PIDs
	droneint := map[uint8]droneConfInternal{}
	//to return
	dronelist := []configs.DroneConfig{}
	for _, drone := range inList {
		droneint[pidc] = droneConfInternal{ManConf: drone, PID: pidc}
		dronelist = append(dronelist, configs.DroneConfig{Address: drone.Address, PID: pidc})
		pidc++
	}

	//TODO compile package
	//we are missing a proper main, so ./hiv should be some other exec
	// err := runComm("zip", []string{"-r", "hiv.zip", "../"}, false)
	//workaround := "/home/dashaylan/go/src/github.com/dashaylan/HiveMind/apps/pi_hivemind.go"
	err := runComm("go", []string{"build", "-o", "hiv"}, true)
	if err != nil {
		log.Fatal("Compilation error: ", err)
		return 0, dronelist, err
	}

	con := configs.Config{IsCBM: false, CBM: GetOutboundIPKvm(), DroneList: dronelist}
	err = configs.WriteConfig("droneConf.json", con)
	if err != nil {
		log.Fatal("Config Creation: ", err)
	}
	fmt.Println("[IPC] StartNodes: Conf file made")

	//allow sshpass to execute
	//sshpass is necessary to execute scp non-interactively
	err = runComm("chmod", []string{"a+x", "./sshpass"}, true)
	if err != nil {
		fmt.Println("[IPC] StartNodes: Unable to grant sshpass necessary permissions,", err)
		return 0, dronelist, err
	}
	fmt.Println("[IPC] StartNodes: Set sshpass permissions")

	fmt.Println("[IPC] StartNodes: Moving onto drone deployment")
	resChan := make(chan configs.DroneConfig, len(inList))
	for _, drone := range droneint {
		go func(droneConf configs.DroneManagerConfig, pid uint8) {
			//ssh into remote machine
			//http://blog.ralch.com/tutorial/golang-ssh-connection/
			fmt.Println("[IPC] StartNodes: Starting deployment for drone", droneConf.Address)
			sshConfig := &ssh.ClientConfig{
				User:            droneConf.Username,
				Auth:            []ssh.AuthMethod{ssh.Password(droneConf.Password)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         15 * time.Second,
			}
			dc := configs.DroneConfig{}

			addrport := droneConf.Address + ":22"
			if droneConf.Port != "" {
				addrport = droneConf.Address + ":" + droneConf.Port
			}
			client, err := ssh.Dial("tcp", addrport, sshConfig)
			if err != nil {
				resChan <- dc
				fmt.Println("[IPC] StartNodes: Unable to set up SSH connection,", err)
				return
			}
			defer client.Close()
			fmt.Println("[IPC] StartNodes: Sucessfully connected")

			//create temp folder
			//rm -rf /tmp/hivemind && mkdir /tmp/hivemind
			//also, kill any older versions of hiv running
			err = remoteComm(client, "kill -9 $(lsof -t -i:"+p2pport+") & rm -rf /tmp/hivemind && mkdir /tmp/hivemind")
			if err != nil {
				fmt.Println("[IPC] StartNodes: Unable to create temp dir,", err)
				resChan <- dc
				return
			}
			fmt.Println("[IPC] StartNodes: Made temp dir")

			//scp file into remote machine
			//scp -q ./hiv. drone2@addr:/tmp/hivemind/hiv
			//, "-P", droneConf.Port
			err = runComm("./sshpass", []string{"-p", droneConf.Password, "scp", "-q", "./hiv", droneConf.Username + "@" + droneConf.Address + ":/tmp/hivemind/hiv"}, true)
			if err != nil {
				fmt.Println("[IPC] StartNodes: Unable to copy exec for drone", droneConf.Address, ",", err)
				resChan <- dc
				return
			}
			fmt.Println("[IPC] StartNodes: Copied exec over")

			//copy over config
			err = runComm("./sshpass", []string{"-p", droneConf.Password, "scp", "-q", "./droneConf.json", droneConf.Username + "@" + droneConf.Address + ":/tmp/hivemind/config.json"}, true)
			if err != nil {
				fmt.Println("[IPC] StartNodes: Unable to copy conf,", err)
				resChan <- dc
				return
			}
			fmt.Println("[IPC] StartNodes: Copied conf over")

			//set permissions
			err = remoteComm(client, "chmod a+x /tmp/hivemind/hiv")
			if err != nil {
				fmt.Println("[IPC] StartNodes: Unable to set permissions,", err)
				resChan <- dc
				return
			}
			fmt.Println("[IPC] StartNodes: Set permissions on exec")

			//execute app with drone settings
			//delay drone startup, to give time for first drone to get into ipc.Init
			err = runComm("/bin/bash", []string{"./backstart.sh", droneConf.Address, droneConf.Username, droneConf.Password}, false)
			if err != nil {
				fmt.Println("[IPC] StartNodes: Unable to run executable,", err)
				resChan <- dc
				return
			}
			fmt.Println("[IPC] StartNodes: Remote drone running")

			//signal as sucessful
			resChan <- configs.DroneConfig{Address: droneConf.Address, PID: pid}
		}(drone.ManConf, drone.PID)
	}

	droneConnected := []configs.DroneConfig{}

	//count number of processes that succeeded
	count := len(inList)
	fmt.Println("[IPC] StartNodes: Waiting for", count, "drones")
	for {
		select {
		case res := <-resChan:
			if res.PID > 0 {
				count--
				fmt.Println("[IPC] StartNodes: Drone", res.Address, "connected! Waiting for", count, "more")
				droneConnected = append(droneConnected, res)

				if count == 0 {
					fmt.Println("[IPC] StartNodes: All drones connected")
					return len(inList), droneConnected, nil
				}
			}
		case <-time.After(10 * time.Second):
			fmt.Println("[IPC] StartNodes: Got", len(inList)-count, "drones, rest timed out")
			return len(inList) - count, droneConnected, nil
		}
	}
}

type connectPeer struct {
	conn net.Conn
	addr string
	pid  uint8
}

//TODO return con object too
func Init(inList []configs.DroneConfig, cbm string) (ipcHandle Ipc, txchan chan []byte, rxchan chan []byte) {
	txchan = make(chan []byte, ipcBufSize)
	rxchan = make(chan []byte, ipcBufSize)
	ipcHandle = Ipc{txChan: txchan, rxChan: rxchan}
	reschan := make(chan connectPeer, len(inList))
	myip := GetOutboundIPKvm()
	fmt.Println("[IPC] Init: starting up")

	// Construct PeerMap
	ipcHandle.peermap = InitPeerMap()

	// Initialize ipcHandle's PID
	for _, drone := range inList {
		if myip == drone.Address {
			//It's me
			ipcHandle.pid = drone.PID
			break
		}
	}

	//start sender goroutine
	go sender(txchan, ipcHandle)

	//start receiver goroutine
	go receiver(rxchan, ipcHandle)

	fmt.Println("[IPC] My CBM is", cbm)
	if cbm == "" {
		//I'm the first drone
		ipcHandle.pid = 0

		locConf, err := configs.ReadDroneConfig()
		if err != nil {
			//can't read from config
			fmt.Println("[IPC] Init: Can't read config as CBM for DroneList")
			log.Fatal(err)
		}
		inList = locConf.DroneList
	} else {
		//give time for CBM to init
		fmt.Println("[IPC] Giving time for CBM to startup")
		select {
		case <-time.After(10 * time.Second):
			fmt.Println("[IPC] Time's up")
		}
		//Dial to CBM (if not CBM)
		//Connect to other drones for peermap (if not cbm)
		//cbm should just wait for connections from other drones
		fmt.Println("[IPC] Connecting to cbm", cbm)
		conn, err := net.Dial("tcp", cbm+p2pport)
		if err != nil {
			//can't connect to CBM
			fmt.Println("[IPC] Unable to connect to CBM")
			ipcHandle.peermap.AddPeer(0, Peer{Address: cbm, Pid: 0})
		} else {
			fmt.Println("[IPC] Connected to CBM", cbm)
			ipcHandle.peermap.AddPeer(0, Peer{Address: cbm, Conn: conn, Pid: 0})

			// Post-Dial Message
			myPID := uint8(0)
			for _, drone := range inList {
				if myip == drone.Address {
					myPID = drone.PID
					break
				}
			}
			mbuf := make([]byte, 3) // destid, srcid, msgid ONLY; Message ID == 50 for post-dial msg
			mbuf[0], mbuf[1], mbuf[2] = uint8(0), myPID, uint8(50)
			// Pass post-dial message to tx channel for sending back
			txchan <- mbuf
		}
	}

	if cbm == "" {
		for _, drone := range inList {
			p := Peer{Address: drone.Address, Pid: drone.PID}
			ipcHandle.peermap.AddPeer(drone.PID, p)
		}
	} else {
		//not the first drone. Connect to peers and self
		for _, drone := range inList {
			if myip == drone.Address {
				//It's me
				ipcHandle.pid = drone.PID
				continue
			}
			go func(drone configs.DroneConfig) {
				conn, err := net.Dial("tcp", drone.Address+p2pport)
				if err != nil {
					//TODO failure to connect
					fmt.Println("[IPC] Init: Failure to connect to", drone.Address)
					reschan <- connectPeer{addr: drone.Address, pid: drone.PID}
					return
				}
				fmt.Println("[IPC] Sucessfully connected to", drone.Address)
				reschan <- connectPeer{conn: conn, addr: drone.Address, pid: drone.PID}
			}(drone)
		}

		count := len(inList) - 1
		for count > 0 {
			timedOut := false

			select {
			case res := <-reschan:
				p := Peer{Conn: res.conn, Address: res.addr, Pid: res.pid}
				ipcHandle.peermap.AddPeer(res.pid, p)
				mbuf := make([]byte, 3) // destid, srcid, msgid ONLY; Message ID == 50 for post-dial msg
				mbuf[0], mbuf[1], mbuf[2] = uint8(res.pid), ipcHandle.pid, uint8(50)
				// Pass post-dial message to tx channel for sending back
				txchan <- mbuf
				count--
				break
			case <-time.After(2 * time.Second):
				//timeout
				timedOut = true
				fmt.Println("[IPC] Init timed out")
				break
			}

			if count == 0 || timedOut {
				// Finished all OR timedOut
				break
			}
		}

		if count > 0 {
			//some failed to connect
			fmt.Println("[IPC] Init: Failed to connect to some drones")
		}
	}

	return ipcHandle, txchan, rxchan
}

//sends messages received on
//tx channel to destination
//connection. Translates
//incoming PID to our IP/Conn
func sender(txchan <-chan []byte, ipchandle Ipc) {
	for {
		select {
		case mes := <-txchan:
			fmt.Println("[Sender] Got something on txchan!")
			dest := mes[0]
			hmpid := mes[1]
			// msgID := mes[2]
			//message := mes[3:]

			// Drone may send message to itself, i.e. drone1 (as drone to as CBM)
			if hmpid == ipchandle.pid && dest == ipchandle.pid {
				fmt.Println("[Sender] Message from myself, redirecting to rxchan")
				ipchandle.rxChan <- mes
			} else {
				// Encode length of message into an array of 4 bytes, BigEndian
				lenMsg := make([]byte, 4)
				binary.BigEndian.PutUint32(lenMsg, uint32(len(mes)))
				//fmt.Println("[Sender] Attempt to send message of length ", lenMsg)
				//fmt.Println("[Sender] Attempt to send message", mes)

				// Prepend the length to the message
				message := append(lenMsg, mes...)

				conn, ex := ipchandle.peermap.GetPeerConn(dest)
				if ex {
					fmt.Println("[Sender] Sending to PID", dest)
					go func() {
						n, e := conn.Write(message)
						fmt.Println("[Sender] Wrote", n, "bytes, error is", e)
					}()
				} else {
					// TODO: Handle unknown PID more gracefully.
					fmt.Println("[ERROR] Sending message to unknown PID: ", dest)
				}
			}
		}
	}
}

//listens to messages on p2pport
//spins off a goroutine to read from
//accepted conns, then
//pushes the []byte to rxchan
func receiver(rxchan chan<- []byte, ipchandle Ipc) {
	laddr, err := net.ResolveTCPAddr("tcp", p2pport)
	if err != nil {
		//Unable to resolve own address
		fmt.Println("[IPC] Receiver: Unable to resolve own address,", err)
		return
	}
	list, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		fmt.Println("[IPC] Receiver: Listen error,", err, laddr)
		return
	}

	fmt.Println("[IPC] Receiver: Ready to accept connections at", GetOutboundIPKvm()+laddr.String())
	for {
		conn, err := list.AcceptTCP()
		fmt.Println("[Receiver] Got something on", p2pport)
		if err != nil {
			fmt.Println("[IPC] Receiver: Failed to accept connection,", err)
			continue
		}
		go rxhandler(rxchan, conn, ipchandle)
	}
}

//TODO: read length first, then only read length amount of bytes
//otherwise, Read blocks waiting for EOF or conn close
func rxhandler(rxchan chan<- []byte, conn *net.TCPConn, ipchandle Ipc) {
	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		// Read length of message
		lenBuf := make([]byte, 4)
		_, err := conn.Read(lenBuf)

		if err != nil && err != io.EOF {
			fmt.Println("[IPC] Rxhandler: Failed to read next bytes (maybe intentionally timeout)", err)

			// Reset timeout (MUST HAVE)
			conn.SetReadDeadline(time.Time{})
			return
		}

		var lenMsg uint32
		bbuf := bytes.NewReader(lenBuf)
		err = binary.Read(bbuf, binary.BigEndian, &lenMsg)
		if err != nil {
			fmt.Println("[IPC] Rxhandler: Failed to retrieve length of message from bytes read,", err)
			return
		}

		if lenMsg > 0 {
			//fmt.Println("[IPC] Rxhandler: Ready to receive ", lenMsg, " bytes of message")

			buf := make([]byte, lenMsg)
			nread, err := conn.Read(buf)
			//fmt.Println("[IPC] Read", nread, "bytes, message is:", buf[:nread])
			//fmt.Println("[IPC] Got message byte array")

			if err != nil && err != io.EOF {
				fmt.Println("[IPC] Rxhandler: Failed to read from connection,", err)
				return
			}

			go rxhelper(rxchan, buf, nread, ipchandle)
		}
	}
}

func rxhelper(rxchan chan<- []byte, buf []byte, nread int, ipchandle Ipc) {
	// Handle the case if it is a post-dial message (Message ID 50)
	// It is less than 4 bytes, so handle before the next if-case
	if buf[2] == 50 {
		fmt.Println("[IPC] Rxhandler: A Post-Dial message, ignored")
		return
	}

	if nread < 4 {
		//headers take 3 bytes, so if less we know something's wrong
		fmt.Println("[IPC] Rxhandler: Malformed message")
		return
	}
	//fmt.Println("[IPC] Rxhandler: I got this packet:", string(buf[:nread]))
	//fmt.Println("[IPC] Rxhandler: I got this packet:", buf[:nread])
	//fmt.Println("[IPC] Got a valid packet")

	// Estabilish a "return" connection to the drone who sent things to me.
	// PID of source drone
	srcID := buf[1]

	p, ex := ipchandle.peermap.GetPeer(srcID)
	if !ex {
		// This drone have never (attempt) to connect the incoming drone before
		fmt.Println("[IPC] Rxhandler: Peermap contains no info about PID", srcID)
		return
	} else if p.Conn != nil {
		// Return if we already have a connection with the incoming drone
		fmt.Println("[IPC] Rxhandler: Peermap contains conn to PID", srcID)

		//send to channel
		rxchan <- buf[:nread]
		return
	}

	fmt.Println("[IPC] Rxhandler: We don't have a connection with Drone with ID", srcID)

	ip := p.Address
	fmt.Println("[IPC] Rxhandler: Found IP", ip, "for Drone", srcID)

	rconn, err := net.Dial("tcp", ip+p2pport)
	if err != nil {
		fmt.Println("[IPC] Rxhandler: Unable to estabilish a return connection to", ip)
	} else {
		fmt.Println("[IPC] Rxhandler: Connected to Drone", srcID, "at", ip)
		ipchandle.peermap.AddPeer(srcID, Peer{Address: ip, Conn: rconn, Pid: srcID})
		mbuf := make([]byte, 3) // destid, srcid, msgid ONLY; Message ID == 50 for post-dial msg
		mbuf[0], mbuf[1], mbuf[2] = srcID, ipchandle.pid, uint8(50)

		// Pass post-dial message to tx channel for sending back
		ipchandle.txChan <- mbuf
	}

	//send to channel
	rxchan <- buf[:nread]
}

// TimeStamp of last connection from drone
var (
	timeLock   sync.Mutex
	timestamps map[string]time.Time
)

// Handler for incoming drone connection to CBM
func HandleIncomingCBMConnection(conn net.Conn) {
	fmt.Println("CBM Connection received")
	// TODO: Update timestamp
}

// Listen for TCP connections from other drones to CBM
func StartCBMServer() (string, error) {
	cbmListener, err := net.Listen("tcp", ":0")
	if err != nil {
		// Something wrong with starting server
		fmt.Println("[ERROR] Can't start CBM server")
		fmt.Println(err.Error())
		return "", err
	}

	go func() {
		for {
			conn, err := cbmListener.Accept()
			if err != nil {
				fmt.Println("[ERROR] Error with incoming connection to CBM, ignored")
				fmt.Println(err.Error())
				continue
			}

			go HandleIncomingCBMConnection(conn)
		}
		cbmListener.Close()
	}()

	fmt.Println("[LOG] CBM Server started: ", cbmListener.Addr().String())
	return cbmListener.Addr().String(), nil
}
