package godht

import (
	"crypto/sha1"
	"io"
	"math/big"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	HasFoundSize = 100000
	TableSize    = 1024
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
	step := big.NewInt(0).Div(num, big.NewInt(cnt+2))
	for i := 1; i < int(cnt+1); i++ {
		item := big.NewInt(0).Mul(step, big.NewInt(int64(i)))
		item.Add(item, big.NewInt(int64(rand.Intn(99))))
		ids = append(ids, item.Bytes())
	}
	return
}

// GenerateId define
func GenerateID() ID {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := sha1.New()
	io.WriteString(hash, time.Now().String())
	io.WriteString(hash, string(random.Int()))
	return hash.Sum(nil)
}

// Neighbor define
func (id ID) Neighbor(tableID ID) ID {
	return append(id[:6], tableID[6:]...)
}
