package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	jsoncomma "github.com/math2001/jsoncomma/internals"
	"github.com/math2001/jsoncomma/server"
)

func main() {

	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
	serverHost := serverCmd.String("host", "localhost", "Address to bind the server to. (if empty, it binds to every interface)")
	serverPort := serverCmd.Int("port", 2442, "The port to listen on")

	if len(os.Args) == 1 {
		serverCmd.Parse(os.Args[1:])
		flag.Usage()
		return
	}

	if os.Args[1] == "server" {
		serverCmd.Parse(os.Args[2:])
		if err := server.Serve(*serverHost, *serverPort); err != nil {
			log.Fatal(err)
		}
		return
	}

	flag.Parse()

	// file/folder names only
	if err := fix(flag.Args()); err != nil {
		log.Fatal(err)
	}
}

func fix(filenames []string) error {
	var wg sync.WaitGroup
	for _, filename := range filenames {
		// I'm not sure about os.O_SYNC. I'm guessing I have to use
		// it because
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			// because we would be reading at the same time as reading
			// from the same file, that means that the read operation and
			// write operation are dependent, which doesn't work with Fixer
			// (it assumes that they are two completely different things)

			// so right now, I'll just do this big fat discusting thing
			// FIXME: is there a nice way to kind of "split" the file,
			// so they have two different carets? (maybe open the file twice?
			// is that possible?)
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Print(err)
			}

			f, err := os.OpenFile(filename, os.O_RDWR|os.O_SYNC, 0644)
			if err != nil {
				log.Print(err)
			}
			defer f.Close()

			if _, err := jsoncomma.Fix(&jsoncomma.Config{}, bytes.NewReader(content), f); err != nil {
				log.Printf("fixing %q: %s", filename, err)
			}
			log.Printf("done %q", filename)
		}(filename)
	}
	wg.Wait()
	return nil
}

func localtest() {
	reader := strings.NewReader(`{
		"hello": "world"
		"this": "is"

		"a": "[1 2 ] test"
	}`)

	fmt.Println(jsoncomma.Fix(&jsoncomma.Config{}, reader, os.Stdout))
}
