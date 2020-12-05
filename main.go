package main

import (
	"bytes"
	"context"
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
	"time"

	jsoncomma "github.com/jsoncomma/jsoncomma/internals"
)

// meta data about the build
var version = "<not specified>"
var commit = "<not specified>"
var date = "<not specified>"

func main() {

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
	printVersion := flag.Bool("version", false, "print the version and exits")

	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "$ jsoncomma server      Starts the optimized server (server -help for more details)")
		fmt.Fprintln(flag.CommandLine.Output(), "$ jsoncomma files...    Fixes all the files")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *printVersion {
		fmt.Println(version, commit, date)
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		// try to see if there is some stuff in stdin

		// try to read from stdin

		stat, err := os.Stdin.Stat()
		if err != nil {
			log.Fatalf("getting os.stdin stat: %s", err)
		}

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

	// this server fix output send on /
	// you can shut it down by getting /shutdown
	// it'll send a reply with a json object {"timedout": <bool>}.
	// if it is true, that means that the server has forcefully
	// shutdown handlers (the timeout is 1 second)

	// This is a very bad implementation. We assume that it is safe
	// to close the handlers on the last handler instruction (wg.Done())
	// instead of when the actual content has been written. In the
	// case of /shutdown, it's closing the channel doneserver. In theory,
	// the corroutine waiting on the channel to be close to shutdown
	// the server could start off straight away, *before the /shutdown
	// handler has even finished*. That would mean that no response is written.
	// As a safety, I flush the data in the handler, to provide some guarantee
	// that the client will recieve the information it expects.

	// this command should try to only output JSON to stdout
	encoder := json.NewEncoder(os.Stdout)

	router := http.NewServeMux()

	server := &http.Server{
		Handler:      router,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		IdleTimeout:  time.Minute,
	}

	// used to keep track of running handlers
	var wg sync.WaitGroup

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		wg.Add(1)
		defer wg.Done()

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

		content, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		body := bytes.NewReader(content)

		defer r.Body.Close()
		if _, err := jsoncomma.Fix(conf, body, w); err != nil {
			log.Printf("fixing: %s", err)
		}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		if err := encoder.Encode(kv{
			"kind": "error",
			"context": "opening socket",
			"error": err.Error(),
			"details": err,
		}); err != nil {
			return err
		}
		return nil
	}

	serverdone := make(chan struct{})

	router.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {

		timedout := WaitTimeout(&wg, 1*time.Second)
		if timedout {
			fmt.Fprintf(w, "{\"timedout\": true}\n")
		} else {
			fmt.Fprintf(w, "{\"timedout\": false}\n")
		}

		// panic if we are not a flusher so that the client gets a big fat error
		// if we can't guarantee that it will get at least one time. This is just
		// to make sure that no client blocks.
		w.(http.Flusher).Flush()

		close(serverdone)
	})

	go func() {
		// I don't know how to guarantee that this will only run once *every* handler
		// has finished. This solution isn't theoratically correct, because this could
		// run as soon as close(serverdone) is called (which is *within* the handler,
		// hence it is not done). In practice, this seems to always work out because
		// it takes the time of concurrent "jump" to close the handler (ie. the handler
		// has enough time to be closed before we get here)
		<-serverdone
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutting down server: %s", err)
			if err := server.Close(); err != nil {
				log.Printf("closing server: %s", err)
			}
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	if err := encoder.Encode(kv{
		"kind": "started",
		"addr": addr.String(),
		"host": addr.IP,
		"port": addr.Port,
	}); err != nil {
		return err
	}

	if err := server.Serve(listener); err != http.ErrServerClosed {
		if err := encoder.Encode(kv{
			"kind":    "error",
			"context": "serving",
			"error":   err.Error(),
			"details": err,
		}); err != nil {
			return err
		}
	} else {
		if err := encoder.Encode(kv{
			"kind": "stopped",
		}); err != nil {
			return err
		}
	}

	return nil
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

// WaitTimeout waits for wg to be done, and takes at most timeout time.
// returns true if it was due to timeout
func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	timer := time.NewTimer(timeout)

	select {
	case <-timer.C:
		return true
	case <-done:
		if !timer.Stop() {
			<-timer.C
		}
		return false
	}
}
