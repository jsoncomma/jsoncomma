package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	jsoncomma "github.com/math2001/jsoncomma/internals"
)

const HeaderTrailing = "X-Trailing"

type kv map[string]interface{}

func Serve(host string, port int) error {
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
