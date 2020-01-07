package main

import "fmt"

func assert(c bool, format string, args ...interface{}) {
	if !c {
		fmt.Printf("[assertion error] "+format+"\n", args...)
		panic("assertion error")
	}
}
