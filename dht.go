package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"net"
	"sync"
	"time"

	"github.com/marksamman/bencode"
	"golang.org/x/time/rate"
)

var seeds = []string{
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"router.utorrent.com:6881",
}

//
type nodeID []byte

// Dht struct description
type Dht struct {
	conn    *net.UDPConn
	limiter *rate.Limiter
	friends chan *node
	exit    chan struct{}
	mu      sync.Mutex
	selfID  nodeID
	secret	[]byte
}

type node struct {
	addr string
	id   string
}

// NewDHT constructor
func NewDHT(addr string, limit int) (*Dht, error) {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}

	d := &Dht{
		conn:    conn.(*net.UDPConn),
		limiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(limit)), limit),
		friends: make(chan *node),
		exit:    make(chan struct{}),
		selfID:  randBytes(20),
		secret:  randBytes(20),
	}

	return d, nil
}

// Run start to run the DHT sniffer
func (d *Dht) Run() {
	d.join()
}

func (d *Dht) join() {
	for _, addr := range seeds {
		d.friends <- &node{
			addr: addr,
			id:   string(randBytes(20)),
		}
	}
}

func (d *Dht) listen() {
	buf := make([]byte, 2048)
	for {
		n, addr, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			close(d.exit)
			return
		}
		d.onMessage(buf[:n], addr)
	}
}

func (d *Dht) send(dict map[string]interface{}, dst *net.UDPAddr) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.conn.WriteToUDP(bencode.Encode(dict), dst)
	return
}

func (d *Dht) onMessage(data []byte, src *net.UDPAddr) {
	dict, err := bencode.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}

	y, ok := dict["y"].(string)
	if !ok {
		return
	}
	switch y {
	case "q":
		d.onQuery(dict, src)
	case "r", "e":
		d.onResponse(dict, src)
	}
}

func (d *Dht) onQuery(dict map[string]interface{}, src *net.UDPAddr) {
	q, ok := dict["q"].(string)
	if !ok {
		return
	}

	switch q {
	case "get_peers":
		d.onGetPeersQuery(dict, src)
	case "announce_peer":
		d.onAnnouncePeerQuery(dict, src)
	case "ping":
		d.onPing(dict, src)
	case "find_node":
		d.onFindNode(dict, src)
	}
}

func (d *Dht) onResponse(dict map[string]interface{}, src *net.UDPAddr) {

}

func (d *Dht) onGetPeersQuery(dict map[string]interface{}, src *net.UDPAddr) {
	tid, ok := dict["t"].(string)
	if !ok {
		return
	}
	a, ok := dict["a"].(map[string]interface{})
	if !ok {
		return
	}
	id, ok := a["id"].(string)
	if !ok {
		return
	}

	r := makeResponse(tid, map[string]interface{}{
		"id": string(d.neighborID(nodeID(id))),
		"nodes" : "",
		"token" : d.makeToken(src),
	})
	d.send(r, src)
}

func (d *Dht) onAnnouncePeerQuery(dict map[string]interface{}, src *net.UDPAddr) {
	a, ok := dict["a"].(map[string]interface{})
	if !ok {
		return
	}

	token, ok := a["token"].(string)
	if !ok || d.validateToken(token, src) {
		return
	}

	infohash, ok := a["info_hash"].(string)
	if !ok {
		return
	}

}

func (d *Dht) onPing(dict map[string]interface{}, src *net.UDPAddr) {

}

func (d *Dht) onFindNode(dict map[string]interface{}, src *net.UDPAddr) {

}

func randBytes(n int) []byte {
	buf := make([]byte, n)
	rand.Read(buf)
	return buf
}

func makeResponse(tid string, res map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": tid,
		"y": "r",
		"r": res,
	}
}

func (d *Dht) neighborID(target nodeID) nodeID {
	id := make([]byte, 20)
	copy(id[:6], target[:6])
	copy(id[6:], d.selfID[6:])
	return id
}

func (d *Dht) makeToken(src *net.UDPAddr) string {
	s := sha1.New()
	s.Write([]byte(src.String()))
	s.Write(d.secret)
	return string(s.Sum(nil))
}

func (d *Dht) validateToken(token string, src *net.UDPAddr) bool {
	return token == d.makeToken(src)
}
