package main

import (
	"fmt"
	"godht/pkg/dht"
)

func main() {
	d, err := dht.NewDHT("0.0.0.0:6882", 500)
	if err != nil {
		fmt.Println(err)
		return
	}
	d.Run()
	for {
		select {
		case a := <-d.Announcements:
			fmt.Println(a)
		}
	}
}
