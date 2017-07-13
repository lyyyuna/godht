package godht

import (
	"fmt"
	"time"
)

func Monitor() {
	for {
		if len(hasfound) >= HasFoundSize {
			mutex.Lock()
			for k := range hasfound {
				delete(hasfound, k)
			}
			hasfound = nil
			hasfound = make(map[string]bool, HasFoundSize)
			mutex.Unlock()
		}
		fmt.Println(len(hasfound))
		time.Sleep(time.Second * 60)
	}
}
