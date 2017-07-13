package godht

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/zeebo/bencode"
)

//BOOTSTRAP define
var BOOTSTRAP = []string{
	"67.215.246.10:6881",
	"212.129.33.50:6881",
	"82.221.103.244:6881"}

// DhtNode init
type DhtNode struct {
	node     *KNode
	table    *KTable
	network  *Network
	krpc     *KRPC
	outQueue chan AnnounceData
}

// NewDhtNode create node
func NewDhtNode(id *ID, outQueue chan AnnounceData, addr string) *DhtNode {
	node := new(KNode)
	node.ID = *id
	dht := new(DhtNode)
	dht.node = node
	dht.table = new(KTable)
	dht.network = NewNetwork(dht, addr)
	dht.krpc = NewKRPC(dht)
	dht.outQueue = outQueue
	return dht
}

// Run dht spider
func (dht *DhtNode) Run() {
	go func() {
		dht.network.Listening()
	}()

	go func() {
		dht.AutofindNode()
	}()

	fmt.Println("One node is running on: %s", dht.network.Conn.LocalAddr().String())
}

// AutofindNode define
func (dhtNode *DhtNode) AutofindNode() {
	if len(dhtNode.table.Nodes) == 0 {
		val := make(map[string]interface{})
		args := make(map[string]string)

		for _, host := range BOOTSTRAP {
			raddr, err := net.ResolveUDPAddr("udp", host)
			if err != nil {
				fmt.Println("Resolve dns error, %s", err)
				return
			}
			node := new(KNode)
			node.Port = raddr.Port
			node.IP = raddr.IP
			node.ID = nil
			dhtNode.FindNode(val, args, node)
		}
	}
	val := make(map[string]interface{})
	args := make(map[string]string)
	for {
		node := dhtNode.table.Pop()
		if node != nil {
			dhtNode.FindNode(val, args, node)
			continue
		}
		time.Sleep(time.Second * 2)
	}

}

// FindNode define
func (dhtNode *DhtNode) FindNode(v map[string]interface{}, args map[string]string, node *KNode) {
	var id ID
	if node.ID != nil {
		id = node.ID.Neighbor(dhtNode.node.ID)
	} else {
		id = dhtNode.node.ID
	}

	v["t"] = fmt.Sprintf("%d", rand.Intn(100))
	v["y"] = "q"
	v["q"] = "find_node"

	args["id"] = string(id)
	args["target"] = string(GenerateID())
	v["a"] = args
	data, err := bencode.EncodeBytes(v)
	if err != nil {
		fmt.Println(err)
		return
	}

	raddr := new(net.UDPAddr)
	raddr.IP = node.IP
	raddr.Port = node.Port
	err = dhtNode.network.Send(data, raddr)
	if err != nil {
		fmt.Println(err)
		return
	}
}
