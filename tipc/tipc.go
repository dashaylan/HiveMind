/*
Package hivemind implements the TreadMarks API.

This file contains the implementation of the test inter-processor messaging
layer. This implements a fast, reliable point-to-point communications
between this node and all other nodes in the DSM.

*** This is implemented quickly to support unit testing of hivemind while
*** production ipc is being developed and tested. This implements just the
*** ability to send/receive messages over a socket without much error handling

*/
package tipc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Tipc interface {
	Connect(ip string, port int) (int, error)
	Close()
}

type IpcConn struct {
	id       uint8
	port     int
	listener *net.TCPListener
	peers    []*peer
	tx, rx   chan []byte
}

type peer struct {
	id   uint8
	ip   string
	port int
	conn *net.TCPConn
}

// NewConnection creates a new Tipc instance.
func NewConnection(port int, id, nrPeer uint8) (*IpcConn, <-chan []byte, chan<- []byte, error) {
	ipc := new(IpcConn)
	ipc.peers = make([]*peer, 1)
	ipc.rx, ipc.tx = make(chan []byte, 100), make(chan []byte, 100)
	ipc.port = port
	ipc.id = id
	ipc.peers = make([]*peer, nrPeer)

	listener, err := net.Listen("tcp", fmt.Sprint(":", port+int(id)))
	if err != nil {
		fmt.Println("Error opening listener", err, port+int(id))
		return nil, nil, nil, err
	}
	ipc.listener = listener.(*net.TCPListener)

	go ipc.listenTask()
	go ipc.sendTask()
	return ipc, ipc.rx, ipc.tx, nil
}

func (ipc *IpcConn) DumpPeers() {
	for i, p := range ipc.peers {
		if p != nil {
			fmt.Printf("P[%d][%d]:%d,%s,%d,%v\n", ipc.id, i, p.id, p.ip, p.port, p.conn)
		}
	}
}

// Connect is called to connect to a another drone=id
func (ipc *IpcConn) Connect(ip string, id uint8) error {
	var err error
	var conn net.Conn

	if id == ipc.id {
		return errors.New("Cannot connect to myself")
	}
	if int(id) > len(ipc.peers)-1 {
		return errors.New("id exceeds configured limit")
	}
	conn, err = net.DialTimeout("tcp", fmt.Sprint(ip, ":", ipc.port+int(id)), time.Second*5)
	for err != nil {
		conn, err = net.DialTimeout("tcp", fmt.Sprint(ip, ":", ipc.port+int(id)), time.Second*5)
	}

	c := conn.(*net.TCPConn)

	writeMsg(conn, []byte{id, 0, 1})

	if err != nil {
		panic("Couldnt dial the host: " + err.Error())
	}

	ipc.peers[id] = &peer{id, ip, ipc.port + int(id), c}

	//go ipc.receiveTask(ipc.peers[id])
	//ipc.DumpPeers()
	return nil
}

func (ipc *IpcConn) Close() {
	close(ipc.tx)
}

// Route outgoing messages either to Rx channel or external connection
func (ipc *IpcConn) sendTask() {
	var id uint8
	for msg := range ipc.tx {
		time.Sleep(0)
		id = msg[0]
		if id == ipc.id {
			ipc.rx <- msg
		} else {
			//fmt.Println("Send[", id, "]", msg)
			writeMsg(ipc.peers[id].conn, msg)
		}
	}
}

// Read messages from connection and put them on the rx channel
func (ipc *IpcConn) receiveTask(peer *peer) {
	//fmt.Println("Starting Rx for", peer.id, " in ", ipc.id)
	for {
		msg := readMsg(peer.conn)
		if msg == nil {
			continue
		}
		ipc.rx <- msg
	}
}

// Listen for incoming connections and setup connection
func (ipc *IpcConn) listenTask() {
	for {
		ipc.listener.SetDeadline(time.Now().Add(time.Millisecond * 500))
		conn, err := ipc.listener.AcceptTCP()
		if err == nil {
			//fmt.Println("Accept:", ipc.id)
			msg := readMsg(conn)
			//fmt.Println("Accept:Msg=", msg)
			if len(msg) != 3 {
				panic("Invalid msg received when connecing")
			}
			id := msg[0]
			ipc.peers[id] = &peer{id, "", 0, conn}
			//ipc.DumpPeers()
			//fmt.Println("Accept: Peers", ipc.peers[id])
			go ipc.receiveTask(ipc.peers[id])
		} else if !strings.HasSuffix(err.Error(), "i/o timeout") {
			panic("unknown error when accepting: " + err.Error())
		} else {
			//fmt.Println("Error Accepting TCP:", err)
		}
	}
}

func writeMsg(conn net.Conn, data []byte) {
	ml := make([]byte, 8)
	binary.PutVarint(ml, int64(len(data)))
	msg := append(ml, data...)
	n := 0
	var err error
	for {
		n, err = conn.Write(msg[n:])
		if err == nil {
			break
		}
		panic(err.Error())
	}
}

func readMsg(conn net.Conn) []byte {
	lbuf := make([]byte, 8)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err := io.ReadFull(conn, lbuf)
	if err != nil {
		return nil
	}
	ml, _ := binary.Varint(lbuf)
	msg := make([]byte, ml)
	_, err = io.ReadFull(conn, msg)
	if err != nil {
		return nil
	}
	return msg
}
