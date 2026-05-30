// Command app is the sample long-running process supervised by the basic
// example. Edit this file while the example is running to see refresh rebuild
// and restart it.
package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("app started — edit examples/basic/app/main.go to trigger a reload")
	for {
		time.Sleep(time.Second)
	}
}
