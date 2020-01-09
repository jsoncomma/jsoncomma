package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	jsoncomma "github.com/math2001/jsoncomma/internals"
)

func main() {

	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
	serverHost := serverCmd.String("host", "localhost", "Address to bind the server to. If empty, it binds to every interface.")
	serverPort := serverCmd.Int("port", 2442, "The port to listen on")

	flag.Usage = func() {
		fmt.Println("$ jsoncomma files...")
		fmt.Println("  Fixes all the files, in place.")
		fmt.Println("$ jsoncoma server")
		fmt.Println("  Runs a server to fix payloads")
		serverCmd.PrintDefaults()
	}

	if len(os.Args) == 1 {
		flag.Usage()
		return
	}

	flag.Parse()

	if os.Args[1] == "server" {
		serverCmd.Parse(os.Args[2:])
		if err := serve(*serverHost, *serverPort); err != nil {
			log.Fatal(err)
		}
		return
	}

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

			// maybe it would be more efficient to write to another file
			// and then delete the original file and rename <other> to <original>
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Print(err)
				return
			}

			f, err := os.OpenFile(filename, os.O_RDWR|os.O_SYNC, 0644)
			if err != nil {
				log.Print(err)
			}
			defer f.Close()

			if _, err := jsoncomma.Fix(&jsoncomma.Config{}, bytes.NewReader(content), f); err != nil {
				log.Printf("fixing %q: %s", filename, err)
				return
			}
		}(filename)
	}
	wg.Wait()
	return nil
}

const HeaderTrailing = "X-Trailing"

type kv map[string]interface{}

func serve(host string, port int) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// FIXME: require X-Protocol-Version?

		if r.URL.Path != "/" {
			respondJSON(w, http.StatusNotFound, kv{
				"kind":         "not found",
				"current path": r.URL.Path,
				"msg":          "should only send requests to /",
			})
			return
		}

		if r.Method != http.MethodPost {
			respondJSON(w, http.StatusMethodNotAllowed, kv{
				"kind":           "Method not allowed",
				"msg":            "should only send POST requests to /",
				"current method": r.Method,
			})
			return
		}

		var trailing bool = false
		if r.Header.Get(HeaderTrailing) == "true" {
			trailing = true
		} else if _, ok := r.Header[HeaderTrailing]; ok && r.Header.Get(HeaderTrailing) != "false" {
			respondJSON(w, http.StatusBadRequest, kv{
				"kind":   "bad request",
				"error":  "bad header value",
				"option": "trailing",
				"header": HeaderTrailing,
				"msg":    fmt.Sprintf("expected 'true', 'false', or not specified, got %v", r.Header[HeaderTrailing]),
			})
			return
		}

		conf := &jsoncomma.Config{
			Trailing: trailing,
			Logs:     nil,
		}

		// we don't actually know if it's JSON. It's just whatever kind of
		// text the user gave us that we passed through some filter
		// the main reason is that the JSON we return may contain
		// comments, trailing comma, etc... Hence it would be wrong to
		// use a application/json header
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		defer r.Body.Close()
		if _, err := jsoncomma.Fix(conf, r.Body, w); err != nil {
			log.Printf("fixing: %s", err)
		}
	})

	return http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)
}

func respondJSON(w http.ResponseWriter, code int, obj kv) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	err := enc.Encode(obj)
	if err != nil {
		log.Printf("respond json: %s", err)
	}
}

func localtest() {
	reader := strings.NewReader(`{
		"hello": "world"
		"this": "is"

		"a": "[1 2 ] test"
	}`)

	fmt.Println(jsoncomma.Fix(&jsoncomma.Config{}, reader, os.Stdout))
}
