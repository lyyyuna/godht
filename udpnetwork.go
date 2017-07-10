package godht

import (
	"fmt"
	"net"
	"time"
)

// Network interface
type Network struct {
	Dht  *DhtNode
	Conn *net.UDPConn
}

// NewNetwork Create a new network interface
func NewNetwork(dhtNode *DhtNode, address string) *Network {
	nw := new(Network)
	nw.Dht = dhtNode

	nw.Init(address)
	return nw
}

// Init it
func (nw *Network) Init(address string) {
	addr := new(net.UDPAddr)
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		panic(err)
	}
	nw.Conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	localaddr := nw.Conn.LocalAddr().(*net.UDPAddr)
	nw.Dht.node.IP = localaddr.IP
	nw.Dht.node.Port = localaddr.Port
}

// Listening on
func (nw *Network) Listening() {
	val := make(map[string]interface{})
	buf := make([]byte, 1024)

	for {
		time.Sleep(10 * time.Millisecond)
		n, raddr, err := nw.Conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		nw.Dht.krpc.Decode(buf[:n], val, raddr)
	}
}

// Send data
func (nw *Network) Send(m []byte, addr *net.UDPAddr) error {
	_, err := nw.Conn.WriteToUDP(m, addr)
	if err != nil {
		fmt.Println(err)
	}
	return err
}
