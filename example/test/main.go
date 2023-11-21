package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 0; i < 10; i++ {
		fmt.Println("Changed me", i)
		time.Sleep(1 * time.Second)
	}
}
