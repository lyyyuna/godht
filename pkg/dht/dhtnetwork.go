package dht

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
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
	conn          *net.UDPConn
	limiter       *rate.Limiter
	friends       chan *node
	exit          chan struct{}
	mu            sync.Mutex
	selfID        nodeID
	secret        []byte
	Announcements chan *announcement
}

type node struct {
	addr string
	id   string
}

type announcement struct {
	Src      *net.UDPAddr
	Infohash string
	Peer     *net.TCPAddr
}

// NewDHT constructor
func NewDHT(addr string, limit int) (*Dht, error) {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		glog.Errorf("Listen on %s failed, error is %v", addr, err)
		return nil, err
	}
	glog.Infof("Listening on %v, the friends rate limit is %v", addr, limit)
	d := &Dht{
		conn:          conn.(*net.UDPConn),
		limiter:       rate.NewLimiter(rate.Every(time.Second/time.Duration(limit)), limit),
		friends:       make(chan *node),
		exit:          make(chan struct{}),
		selfID:        randBytes(20),
		secret:        randBytes(20),
		Announcements: make(chan *announcement),
	}

	return d, nil
}

// Run start to run the DHT sniffer
func (d *Dht) Run() {
	fmt.Println("Begin to run....")
	glog.Info("Begin to run...")
	go d.join()
	go d.listen()
	go d.makeFriends()
}

func (d *Dht) join() {
	for i := 0; i < 3; i++ {
		for _, addr := range seeds {
			d.friends <- &node{
				addr: addr,
				id:   string(randBytes(20)),
			}
		}
	}
}

func (d *Dht) listen() {
	buf := make([]byte, 2048)
	for {
		n, addr, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			//close(d.exit)
			//return
			glog.Errorf("Read from UDP port failed, %v", err)
			continue
		}
		d.onMessage(buf[:n], addr)
	}
}

func (d *Dht) makeFriends() {
	for {
		select {
		case node := <-d.friends:
			d.findNode(node.addr, nodeID(node.id))
		case <-d.exit:
			return
		}
	}
}

func (d *Dht) send(dict map[string]interface{}, dst *net.UDPAddr) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.conn.WriteToUDP(bencode.Encode(dict), dst)
	return
}

func (d *Dht) findNode(dst string, target nodeID) {
	q := makeRequest(string(randBytes(2)), "find_node",
		map[string]interface{}{
			"id":     string(d.neighborID(target)),
			"target": string(randBytes(20)),
		})
	addr, err := net.ResolveUDPAddr("udp", dst)
	if err != nil {
		glog.Errorf("Fail to reslove destination UDP addr, the error is: %v", err)
		return
	}

	go d.send(q, addr)
}

func (d *Dht) onMessage(data []byte, src *net.UDPAddr) {
	dict, err := bencode.Decode(bytes.NewReader(data))
	if err != nil {
		glog.Errorf("Fail to bencode decode, error is: %v", err)
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
	fmt.Println(q)
	switch q {
	case "get_peers":
		d.onGetPeersQuery(dict, src)
	case "announce_peer":
		a, ok := d.onAnnouncePeerQuery(dict, src)
		if ok {
			d.Announcements <- a
		}
	case "ping":
		d.onPing(dict, src)
	case "find_node":
		d.onFindNode(dict, src)
	}
}

func (d *Dht) onResponse(dict map[string]interface{}, src *net.UDPAddr) {
	r, ok := dict["r"].(map[string]interface{})
	if !ok {
		return
	}

	nodes, ok := r["nodes"].(string)
	if !ok {
		return
	}

	length := len(nodes)
	if length%26 != 0 {
		return
	}

	for i := 0; i < length; i += 26 {
		if !d.limiter.Allow() {
			continue
		}
		id := nodes[i : i+20]
		ip := net.IP([]byte(nodes[i+20 : i+24])).String()
		port := binary.BigEndian.Uint16([]byte(nodes[i+24 : i+26]))

		addr := ip + ":" + strconv.Itoa(int(port))

		d.friends <- &node{addr: addr, id: id}
	}
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
	infohash, ok := a["info_hash"].(string)
	fmt.Println(hex.EncodeToString([]byte(infohash)))
	r := makeResponse(tid, map[string]interface{}{
		"id":    string(d.neighborID(nodeID(id))),
		"nodes": "",
		"token": d.makeToken(src),
	})
	go d.send(r, src)
}

func (d *Dht) onAnnouncePeerQuery(dict map[string]interface{}, src *net.UDPAddr) (*announcement, bool) {
	a, ok := dict["a"].(map[string]interface{})
	if !ok {
		return nil, false
	}

	token, ok := a["token"].(string)
	if !ok || d.validateToken(token, src) {
		return nil, false
	}

	infohash, ok := a["info_hash"].(string)
	if !ok {
		return nil, false
	}

	// check port
	port := int64(src.Port)
	impliedPort, ok := a["implied_port"].(int64)
	if !ok {
		return nil, false
	}
	if impliedPort == 0 {
		if p, ok := a["port"].(int64); ok {
			port = p
		}
	}

	return &announcement{
		Src:      src,
		Infohash: infohash,
		Peer:     &net.TCPAddr{IP: src.IP, Port: int(port)},
	}, true

}

func (d *Dht) onPing(dict map[string]interface{}, src *net.UDPAddr) {
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
	if len(id) == 20 {
		r := makeResponse(tid, map[string]interface{}{
			"id": string(d.neighborID(nodeID(id))),
		})
		go d.send(r, src)
	} else {
		r := makeResponse(tid, map[string]interface{}{
			"id": string(nodeID(id)),
		})
		go d.send(r, src)
	}
}

func (d *Dht) onFindNode(dict map[string]interface{}, src *net.UDPAddr) {
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
	if len(id) == 20 {
		r := makeResponse(tid, map[string]interface{}{
			"id":    string(d.neighborID(nodeID(id))),
			"nodes": "",
		})
		go d.send(r, src)
	}
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

func makeRequest(tid string, q string, a map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": tid,
		"y": "q",
		"q": q,
		"a": a,
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
