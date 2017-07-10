package godht

import (
	"fmt"
)

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
	return dht
}

// Run dht spider
func (dht *DhtNode) Run() {
	go func() {
		dht.network.Listening()
	}()

	go func() {

	}()

	fmt.Println("One node is running on: %s", dht.network.Conn.LocalAddr().String())
}
