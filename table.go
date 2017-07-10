package godht

import "net"
import "sync"
import "math/big"

const (
	HasFoundSize = 100000
)

var mutex sync.Mutex
var hasfound = make(map[string]bool, HasFoundSize)

// ID define
type ID []byte

// KNode define
type KNode struct {
	ID   ID
	IP   net.IP
	Port int
}

// KTable define
type KTable struct {
	Nodes []*KNode
	cap   int64

	Snodes []*KNode
}

// Put node into a table
func (table *KTable) Put(node *KNode) {
	ids := string([]byte(node.ID))
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := hasfound[ids]; !ok {
		table.Nodes = append(table.Nodes, node)
	}
	if len(table.Snodes) < 8 {
		table.Snodes = append(table.Snodes, node)
	}
}

// Pop out a node
func (table *KTable) Pop() *KNode {
	if len(table.Nodes) > 0 {
		n := table.Nodes[0]
		table.Nodes = table.Nodes[1:]
		return n
	}
	return nil
}

// GenerateIDList for start
func GenerateIDList(cnt int64) (ids []ID) {
	max := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	num := big.NewInt(0).SetBytes(max)
	step := big.NewInt(0).Div(num, big.NewInt(coubt+2))

}
