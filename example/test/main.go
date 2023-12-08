package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 0; i < 100; i++ {
		fmt.Println("changeme", i)
		time.Sleep(1 * time.Second)
	}
}
