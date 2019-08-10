package main

import "fmt"

func main() {
	dht, err := NewDHT("0.0.0.0:6882", 500)
	if err != nil {
		fmt.Println(err)
		return
	}
	dht.Run()
	for {
		select {
		case a := <-dht.announcements:
			fmt.Println(a)
		}
	}
}
