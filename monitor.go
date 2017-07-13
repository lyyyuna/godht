package godht

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	l = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
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

func logger(v ...interface{}) {
	l.Println(v)
}
