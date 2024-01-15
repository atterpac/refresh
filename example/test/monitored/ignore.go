package main

import (
	"fmt"
)

// This file while in a monitored folder with a monitored extension carries the same name as an ignore.go in the ignored files
// Changes to this file will be recognized in the debug logs but will not trigger a reload
func ignored() {
	fmt.Println("This file is ignored and not changed")

}

