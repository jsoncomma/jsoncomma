package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	jsoncomma "github.com/math2001/jsoncomma/internals"
)

const json = `{
	"hello": "world"
	// comment
	"this is": ["a" "test" true "right?" 2]
}
`

func main() {
	var buffer bytes.Buffer
	config := &jsoncomma.Config{
		Trailling: false,
	}
	fmt.Printf("in: %d bytes\n", len(json))
	n, err := jsoncomma.Fix(config, strings.NewReader(json), &buffer)
	if err != nil {
		log.Printf("error: %s", err)
	}
	fmt.Println("wrote", n, "bytes")
	out := buffer.String()
	fmt.Println(out)
	assert(len(out) == n, "wrote %d bytes, expected %d", len(out), n)
}
