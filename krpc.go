package godht

import (
	"net"

	"bytes"

	"github.com/zeebo/bencode"
)

type action func(arg map[string]interface{}, raddr *net.UDPAddr)

// KRPC define
type KRPC struct {
	Dht   *DhtNode
	Types map[string]action
	tid   uint32
}

//KRPCMessage define
type KRPCMessage struct {
	T      string
	Y      string
	Addion interface{}
	Addr   *net.UDPAddr
}

// Query define
type Query struct {
	Y string
	A map[string]interface{}
}

// Response define
type Response struct {
	R map[string]interface{}
}

// AnnounceData define
type AnnounceData struct {
	Infohash    string
	IP          net.IP
	Port        int
	ImpliedPort int
}

// Decode message, use bencode
func (krpc *KRPC) Decode(data []byte, val map[string]interface{}, raddr *net.UDPAddr) error {
	if err := bencode.DecodeBytes(data, &val); err != nil {
		return err
	}

	message := new(KRPCMessage)

	var ok bool
	message.T, ok = val["t"].(string)
	if !ok {
		return nil
	}

	message.Y, ok = val["y"].(string)
	if !ok {
		return nil
	}

	message.Addr = raddr

	switch message.Y {
	case "q":
		query := new(Query)
		if q, ok := val["q"].(string); ok {
			query.Y = q
		} else {
			return nil
		}
		if a, ok := val["a"].(map[string]interface{}); ok {
			query.A = a
			message.Addion = query
		} else {
			return nil
		}
	case "r":
		res := new(Response)
		if r, ok := val["r"].(map[string]interface{}); ok {
			res.R = r
			message.Addion = res
		} else {
			return nil
		}
	default:
		return nil
	}

	switch message.Y {
	case "q":
		krpc.Query(message)
	case "r":
		krpc.Response(message)
	}
	return nil
}

// NewKRPC define
func NewKRPC(dhtNode *DhtNode) *KRPC {
	krpc := new(KRPC)
	krpc.Dht = dhtNode

	return krpc
}

// Response message
func (krpc *KRPC) Response(msg *KRPCMessage) {
	if len(krpc.Dht.table.Nodes) <= TableSize &&
		len(hasfound) < HasFoundSize {
		if response, ok := msg.Addion.(*Response); ok {
			if nodestr, ok := response.R["nodes"].(string); ok {
				nodes := parseBytesToNodes([]byte(nodestr))
				for _, node := range nodes {
					if node.Port > 0 && node.Port <= (1<<16) {
						krpc.Dht.table.Put(node)
					}
				}
			}
		}
	}
}

// Query message
func (krpc *KRPC) Query(msg *KRPCMessage) {
	if query, ok := msg.Addion.(*Query); ok {
		if query.Y == "get_peers" {
			if infohash, ok := query.A["info_hash"].(string); ok {
				if len(infohash) != 20 {
					return
				}
				if msg.T == "" {
					return
				}
				fromID, ok := query.A["id"].(string)
				if !ok {
					return
				}
				if len(fromID) != 20 {
					return
				}

				result := AnnounceData{}
				result.Infohash = ID(infohash).String()
				result.IP = msg.Addr.IP

				// print infohash
				krpc.Dht.outQueue <- result

				nodes := convertBytesStream(krpc.Dht.table.Snodes)
				data, _ := krpc.encodeNodeResult(msg.T, "asdf13e", nodes)
				go krpc.Dht.network.Send(data, msg.Addr)
			}
		}
		if query.Y == "announce_peer" {
			if infohash, ok := query.A["info_hash"].(string); ok {
				token, ok := query.A["token"].(string)
				if !ok {
					return
				} else if token != "asdf13e" {
					return
				}

				var port int
				var impliedPort int64
				impliedPort, ok = query.A["implied_port"].(int64)
				if ok {
					if impliedPort != 0 {
						port = msg.Addr.Port
					}
				} else {
					pport, ok := query.A["port"].(int64)
					if ok {
						port = int(pport)
					}
				}

				result := AnnounceData{}
				result.Infohash = ID(infohash).String()
				result.IP = msg.Addr.IP
				result.Port = port
				result.ImpliedPort = int(impliedPort)

				var data []byte
				if id, ok := query.A["id"].(string); ok {
					newID := Neightor(id, krpc.Dht.node.ID.String())
					data, _ = krpc.encodeNormalResult(msg.T, newID)
				} else {
					data, _ = krpc.encodeNormalResult(msg.T, krpc.Dht.node.ID.String())
				}
				go krpc.Dht.network.Send(data, msg.Addr)
			}
		}

		if query.Y == "ping" {
			var data []byte
			if id, ok := query.A["id"].(string); ok && len(id) == 20 {
				newID := Neightor(id, krpc.Dht.node.ID.String())
				data, _ = krpc.encodeNormalResult(msg.T, newID)
			} else {
				data, _ = krpc.encodeNormalResult(msg.T, krpc.Dht.node.ID.String())
			}
			go krpc.Dht.network.Send(data, msg.Addr)
		}

		if query.Y == "find_node" {
			if msg.T == "" {
				return
			}
			nodes := convertBytesStream(krpc.Dht.table.Snodes)
			data, _ := krpc.encodeNodeResult(msg.T, "", nodes)
			go krpc.Dht.network.Send([]byte(data), msg.Addr)
		}
	}
}

// parseBytesToNodes define
func parseBytesToNodes(data []byte) []*KNode {
	var nodes []*KNode
	for i := 0; i < len(data); i = i + 26 {
		if i+26 > len(data) {
			break
		}

		kn := data[i : i+26]
		node := new(KNode)
		node.ID = ID(kn[0:20])
		node.IP = kn[20:24]
		port := kn[24:26]
		node.Port = int(port[0])<<8 + int(port[1])
		nodes = append(nodes, node)
	}
	return nodes
}

// conver
func convertBytesStream(nodes []*KNode) []byte {
	buf := bytes.NewBuffer(nil)
	for _, v := range nodes {
		convertNodeInfo(buf, v)
	}
	return buf.Bytes()
}

func convertNodeInfo(buf *bytes.Buffer, v *KNode) {
	buf.Write(v.ID)
	convertIPPort(buf, v.IP, v.Port)
}

func convertIPPort(buf *bytes.Buffer, ip net.IP, port int) {
	buf.Write(ip.To4())
	buf.WriteByte(byte((port & 0xFF00) >> 8))
	buf.WriteByte(byte(port & 0xFF))
}

func (krpc *KRPC) encodeNodeResult(tid string, token string, nodes []byte) ([]byte, error) {
	v := make(map[string]interface{})
	defer func() { v = nil }()
	v["t"] = tid
	v["y"] = "r"
	args := make(map[string]string)
	defer func() { args = nil }()
	args["id"] = string(krpc.Dht.node.ID)
	if token != "" {
		args["token"] = token
	}
	args["nodes"] = bytes.NewBuffer(nodes).String()
	v["r"] = args
	return bencode.EncodeBytes(v)
}

func (krpc *KRPC) encodeNormalResult(tid string, id string) ([]byte, error) {
	v := make(map[string]interface{})
	defer func() { v = nil }()
	v["t"] = tid
	v["y"] = "r"
	args := make(map[string]string)
	defer func() { args = nil }()
	args["id"] = id
	v["r"] = args
	return bencode.EncodeBytes(v)
}
