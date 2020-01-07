package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
)

const json = `{
	"hello": "world"
	// comment
	"this is": ["a" "test" true "right?" 2]
}
`

func main() {
	var buffer bytes.Buffer
	config := &Config{
		Trailling: false,
	}
	fmt.Printf("in: %d bytes\n", len(json))
	n, err := Fix(config, strings.NewReader(json), &buffer)
	if err != nil {
		log.Printf("error: %s", err)
	}
	fmt.Println("wrote", n, "bytes")
	out := buffer.String()
	fmt.Println(out)
	assert(len(out) == n, "wrote %d bytes, expected %d", len(out), n)
}
