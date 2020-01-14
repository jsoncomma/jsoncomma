package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	jsoncomma "github.com/jsoncomma/jsoncomma/internals"
)

func main() {

	// try to read from stdin

	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("getting os.stdin stat: %s", err)
	}


	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
	serverHost := serverCmd.String("host", "localhost", "Address to bind the server to. \nIf empty, it binds to every interface.")
	// note here that we have to explicitely write the "default 0" because go thinks we don't care
	// since 0 is the nil value of an int
	serverPort := serverCmd.Int("port", 0, "The port to listen on.\n0 means 'chose random unused one' (default 0)")

	serverCmd.Usage = func() {
		fmt.Fprintln(serverCmd.Output(), "Runs an optimized web server to fix payloads")
		serverCmd.PrintDefaults()
	}

	tostdout := flag.Bool("stdout", false, "write to stdout instead of in place")

	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "$ jsoncomma files...    Fixes all the files")
		fmt.Fprintln(flag.CommandLine.Output(), "$ jsoncomma server      Starts the optimized server (server -help for more details)")
	}

	if len(os.Args) == 1 {
		// try to see if there is some stuff in stdin
		if stat.Mode()&os.ModeCharDevice == 0 {
			if len(os.Args) > 1 {
				log.Fatal("can't handle arguments when piping from stdin")
			}
			if _, err := jsoncomma.Fix(&jsoncomma.Config{}, os.Stdin, os.Stdout); err != nil {
				log.Fatal(err)
			}
			return
		} else {
			// print the help
			flag.Usage()
		}
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
	if err := fix(flag.Args(), *tostdout); err != nil {
		log.Fatal(err)
	}
}

func fix(filenames []string, tostdout bool) error {
	var wg sync.WaitGroup

	config := &jsoncomma.Config{}

	for _, filename := range filenames {
		// I'm not sure about os.O_SYNC. I'm guessing I have to use
		// it because

		if tostdout {
			f, err := os.Open(filename)
			if err != nil {
				log.Print(err)
				continue
			}
			if _, err := jsoncomma.Fix(config, f, os.Stdout); err != nil {
				log.Printf("fixing %q: %s", filename, err)
				continue
			}
		} else {
			wg.Add(1)
			go func(config *jsoncomma.Config, filename string) {
				defer wg.Done()
				if err := fixfile(config, filename); err != nil {
					log.Println(err)
					return
				}
			}(config, filename)
		}

	}
	wg.Wait()
	return nil
}

// fixfile fixes the file in place
func fixfile(config *jsoncomma.Config, filename string) error {
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
		return err
	}

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := jsoncomma.Fix(config, bytes.NewReader(content), f); err != nil {
		return fmt.Errorf("fixing %q: %s", filename, err)
	}
	return nil
}

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

		conf := &jsoncomma.Config{
			Logs: nil,
		}

		// we don't actually know if it's JSON. It's just whatever kind of
		// text the user gave us that we passed through some filter
		// the main reason is that the JSON we return may contain
		// comments etc... Hence it would be wrong to
		// use a application/json header
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		defer r.Body.Close()
		if _, err := jsoncomma.Fix(conf, r.Body, w); err != nil {
			log.Printf("fixing: %s", err)
		}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}

	// output in JSON just to make it easy to parse
	enc := json.NewEncoder(os.Stdout)
	addr := listener.Addr().(*net.TCPAddr)
	enc.Encode(kv{
		"addr": addr.String(),
		"host": addr.IP,
		"port": addr.Port,
	})
	return http.Serve(listener, nil)
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
