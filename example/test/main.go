package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)
		fmt.Println("changed me", i)
		time.Sleep(1 * time.Second)
	}
}
