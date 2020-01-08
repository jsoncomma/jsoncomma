package jsoncomma_test

import (
	"bytes"
	"strings"
	"testing"

	jsoncomma "github.com/math2001/jsoncomma/internals"
)

func TestAddCommas(t *testing.T) {
	table := []struct {
		in string
		// the expected output that is pure valid JSON
		valid string
		// the expected output with trailling option on
		trailling string
	}{
		{
			in:        `{ "hello": "world" "oops": "world" }`,
			valid:     `{ "hello": "world", "oops": "world" }`,
			trailling: `{ "hello": "world", "oops": "world", }`,
		},
		{
			in:        `{ "hello": 2 "oops": "test" }`,
			valid:     `{ "hello": 2, "oops": "test" }`,
			trailling: `{ "hello": 2, "oops": "test", }`,
		},
		{
			in: `["a" 2 4
			{"nested": "keys"
				"weird":"whitespace"}
			["still" "works"]]`,
			valid: `["a", 2, 4,
			{"nested": "keys",
				"weird":"whitespace"},
			["still", "works"]]`,
			trailling: `["a", 2, 4,
			{"nested": "keys",
				"weird":"whitespace",},
			["still", "works"],]`,
		},
		{
			in: `["a" 2 4
			// with comments!
			{"nested": "keys" // kind of cool
				"weird":"whitespace"}
			["still" "works"]
			// i like it
			]`,
			valid: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace"},
			["still", "works"]
			// i like it
			]`,
			trailling: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace",},
			["still", "works"],
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
			valid: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace, and sneaky [1 2]"},
			["still", "works"]
			// i like it
			]`,
			trailling: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace, and sneaky [1 2]"},
			["still", "works"]
			// i like it
			]`,
		},
		{
			in:        `{"hello": "world" "oops": "world"}`,
			valid:     `{"hello": "world", "oops": "world"}`,
			trailling: `{"hello": "world", "oops": "world",}`,
		},
		{
			in: `{"hello": "world"
			"oops": "world"}`,
			valid: `{"hello": "world",
			"oops": "world"}`,
			trailling: `{"hello": "world",
			"oops": "world",}`,
		},
		{
			in: `{"hello": "world"
			"oops": ["nested"
			 "keys" "inline"]}`,
			valid: `{"hello": "world",
			"oops": ["nested",
			 "keys", "inline"]}`,
			trailling: `{"hello": "world",
			"oops": ["nested",
			 "keys", "inline",],}`,
		},

		// thanks fuzzing :-)
		{
			in:        "0\t",
			valid:     "0\t",
			trailling: "0\t",
		},
		{
			in:        `""`,
			valid:     `""`,
			trailling: `""`,
		},
		{
			in:        "60",
			valid:     "60",
			trailling: "60",
		},
		{
			in:        "/[",
			valid:     "/[",
			trailling: "/[",
		},
		{
			in:        "0",
			valid:     "0",
			trailling: "0",
		},
	}

	for _, row := range table {
		for _, trailling := range []bool{false} {
			var logs bytes.Buffer

			config := &jsoncomma.Config{
				Trailling: trailling,
				Logs:      &logs,
			}

			var actual bytes.Buffer
			actual.Grow(len(row.valid))

			written, err := jsoncomma.Fix(config, strings.NewReader(row.in), &actual)
			if err != nil {
				t.Logf("logs\n%s", logs.String())
				t.Errorf("in: %#q, err: %s", row.in, err)
			}

			var expected string
			if trailling {
				expected = row.trailling
			} else {
				expected = row.valid
			}

			actualString := actual.String()
			if int64(len(actualString)) != written {
				t.Logf("logs\n%s", logs.String())
				t.Errorf("in: %#q, output: %#q (%d bytes), yet written %d bytes", row.in, actualString, len(actualString), written)
			}
			if actualString != expected {
				t.Logf("logs\n%s", logs.String())
				t.Errorf("in: %#q\nactual:   %#q\nexpected: %#q", row.in, actualString, expected)
			}
		}

	}
}

// TODO: fuzz (make it's parsable json, pass through AddComma, and make sure you get equivalent json out)
// TODO: fuzz random stuff and make sure that the len(out) >= len(in), and diff is only commas
