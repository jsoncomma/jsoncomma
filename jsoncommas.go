package main

import (
	"fmt"
	"io"
)


func main() {
	fmt.Println("hello world")
}

type Config struct {
	Trailling bool
}

func AddCommas(config *Config, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	for {
		
	}
	return nil
}