package jsoncomma_test

import (
	"bytes"
	"strings"
	"testing"

	jsoncomma "github.com/math2001/jsoncomma/internals"
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

		// thanks fuzzing :-)
		{
			in:  "0\t",
			out: "0\t",
		},
		{
			in:  `""`,
			out: `""`,
		},
		{
			in: "60",
			out: "60",
		},
		{
			in: "/[",
			out: "/[",
		},
	}

	for _, row := range table {
		var logs bytes.Buffer

		config := &jsoncomma.Config{
			Trailling: false,
			Logs:      &logs,
		}

		var actual bytes.Buffer
		actual.Grow(len(row.out))

		written, err := jsoncomma.Fix(config, strings.NewReader(row.in), &actual)
		if err != nil {
			t.Logf("logs\n%s", logs.String())
			t.Errorf("in: %#q, err: %s", row.in, err)
		}

		actualString := actual.String()
		if int64(len(actualString)) != written {
			t.Logf("logs\n%s", logs.String())
			t.Errorf("in: %#q, output: %#q (%d bytes), yet written %d bytes", row.in, actualString, len(actualString), written) 
		}
		if actualString != row.out {
			t.Logf("logs\n%s", logs.String())
			t.Errorf("in: %#q\nactual:   %#q\nexpected: %#q", row.in, actualString, row.out)
		}

	}
}

// TODO: fuzz (make it's parsable json, pass through AddComma, and make sure you get equivalent json out)
// TODO: fuzz random stuff and make sure that the len(out) >= len(in), and diff is only commas
