package main

import (
	"fmt"
	"godht"
)

var (
	resChan       = make(chan godht.AnnounceData, 100)
	nodeNum int64 = 10
)

func main() {
	idlist := godht.GenerateIDList(nodeNum)
	for k, id := range idlist {
		go func(port int, id godht.ID) {
			dhtNode := godht.NewDhtNode(&id, resChan, fmt.Sprintf(":%v", 20000+port))
			dhtNode.Run()
		}(k, id)
	}

	go godht.Monitor()

	for {
		select {
		case hashID := <-resChan:
			fmt.Println("magnet:?xt=urn:btih:" + hashID.Infohash)
		}
	}
}
