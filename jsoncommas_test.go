package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAddCommas(t *testing.T) {
	table := []struct {
		in  string
		out string
	}{
		{
			in:  `{ "hello": "world" "oops": "world" }`,
			out: `{ "hello": "world", "oops": "world" }`,
		},
		{
			in:  `{ "hello": 2 "oops": "test" }`,
			out: `{ "hello": 2, "oops": "test" }`,
		},
		{
			in: `["a" 2 4
			{"nested": "keys"
				"weird":"whitespace"}
			["still" "works"]]`,
			out: `["a", 2, 4,
			{"nested": "keys",
				"weird":"whitespace"},
			["still", "works"]]`,
		},
		{
			in: `["a" 2 4
			// with comments!
			{"nested": "keys" // kind of cool
				"weird":"whitespace"}
			["still" "works"]
			// i like it
			]`,
			out: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace"},
			["still", "works"]
			// i like it
			]`,
		},
		{
			in: `["a" 2 4
			// with comments!
			{"nested": "keys" // kind of cool
				"weird":"whitespace, and sneaky [1 2]"}
			["still" "works"]
			// i like it
			]`,
			out: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace, and sneaky [1 2]"},
			["still", "works"]
			// i like it
			]`,
		},
		{
			in:  `{"hello": "world" "oops": "world"}`,
			out: `{"hello": "world", "oops": "world"}`,
		},
		{
			in: `{"hello": "world"
			"oops": "world"}`,
			out: `{"hello": "world",
			"oops": "world"}`,
		},
		{
			in: `{"hello": "world"
			"oops": ["nested"
			 "keys" "inline"]}`,
			out: `{"hello": "world",
			"oops": ["nested",
			 "keys", "inline"]}`,
		},
	}
	config := &Config{
		Trailling: false,
	}

	t.Logf("config: %v", config)

	for _, row := range table {
		var actual bytes.Buffer
		actual.Grow(len(row.out))

		if _, err := Fix(config, strings.NewReader(row.in), &actual); err != nil {
			t.Errorf("in: %s, err: %s", row.in, err)
		}

		actual_str := actual.String()
		if actual_str != row.out {
			t.Errorf("in: %q\nactual:   %q\nexpected: %q", row.in, actual_str, row.out)
		}

	}
}

// TODO: fuzz (make it's parsable json, pass through AddComma, and make sure you get equivalent json out)
// TODO: fuzz random stuff and make sure that the len(out) >= len(in), and diff is only commas
