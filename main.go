package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	jsoncomma "github.com/math2001/jsoncomma/internals"
)

const HeaderTrailing = "X-Trailing"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			respondJSON(w, http.StatusBadRequest, kv{
				"kind": "bad request",
				"msg":  "should only send requests to /",
			})
			return
		}

		if r.Method != http.MethodPost {
			respondJSON(w, http.StatusBadRequest, kv{
				"kind":           "invalid method",
				"msg":            "should only send POST requests to /",
				"current method": r.Method,
			})
			return
		}

		var trailing bool
		if r.Header.Get(HeaderTrailing) == "true" {
			trailing = true
		} else if _, ok := r.Header[HeaderTrailing]; ok {
			respondJSON(w, http.StatusBadRequest, kv{
				"kind":   "bad request",
				"error":  "bad option",
				"option": "trailing",
				"msg":    fmt.Sprintf("expected 'true' or nothing, got %v", r.Header[HeaderTrailing]),
			})
			return
		}

		conf := &jsoncomma.Config{
			Trailing: trailing,
			Logs:     os.Stderr,
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

	// 2442 comes from sum(map(lambda c: ord(c), "json")) == 442. It's too small
	// so 2442 is cool because it reads backwards
	err := http.ListenAndServe(":2442", nil)
	if err != nil {
		log.Fatal(err)
	}
}

// key value
type kv map[string]interface{}

type resp struct {
	code int
	body kv
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
