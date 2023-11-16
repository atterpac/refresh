package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 0; i < 100; i++ {
		time.Sleep(1*time.Second)
		fmt.Println("Println # ", i)
	}
}
