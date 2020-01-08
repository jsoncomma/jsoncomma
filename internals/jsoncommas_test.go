package jsoncomma_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	jsoncomma "github.com/math2001/jsoncomma/internals"
)

func TestAddCommas(t *testing.T) {
	t.Parallel()
	table := []struct {
		in string
		// the expected output that is pure valid JSON
		valid string
		// the expected output with trailing option on
		trailing string
	}{
		{
			in:       `{ "hello": "world" "oops": "world" }`,
			valid:    `{ "hello": "world", "oops": "world" }`,
			trailing: `{ "hello": "world", "oops": "world", }`,
		},
		{
			in:       `{ "hello": 2 "oops": "test" }`,
			valid:    `{ "hello": 2, "oops": "test" }`,
			trailing: `{ "hello": 2, "oops": "test", }`,
		},
		{
			in:       "{ \"hello\": 2\n\"oops\": \"test\" }",
			valid:    "{ \"hello\": 2,\n\"oops\": \"test\" }",
			trailing: "{ \"hello\": 2,\n\"oops\": \"test\", }",
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
			trailing: `["a", 2, 4,
			{"nested": "keys",
				"weird":"whitespace",},
			["still", "works",],]`,
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
			trailing: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace",},
			["still", "works",],
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
			trailing: `["a", 2, 4,
			// with comments!
			{"nested": "keys", // kind of cool
				"weird":"whitespace, and sneaky [1 2]",},
			["still", "works",],
			// i like it
			]`,
		},
		{
			in:       `{"hello": "world" "oops": "world"}`,
			valid:    `{"hello": "world", "oops": "world"}`,
			trailing: `{"hello": "world", "oops": "world",}`,
		},
		{
			in: `{"hello": "world"
			"oops": "world"}`,
			valid: `{"hello": "world",
			"oops": "world"}`,
			trailing: `{"hello": "world",
			"oops": "world",}`,
		},
		{
			in: `{"hello": "world"
			"oops": ["nested"
			 "keys" "inline"]}`,
			valid: `{"hello": "world",
			"oops": ["nested",
			 "keys", "inline"]}`,
			trailing: `{"hello": "world",
			"oops": ["nested",
			 "keys", "inline",],}`,
		},
		{
			in:       `{"a": "b""c": "d"}`,
			valid:    `{"a": "b","c": "d"}`,
			trailing: `{"a": "b","c": "d",}`,
		},
		{
			in:       `[true true 123 false true]`,
			valid:    `[true, true, 123, false, true]`,
			trailing: `[true, true, 123, false, true,]`,
		},

		// thanks fuzzing :-)
		{
			in:       "0\t",
			valid:    "0\t",
			trailing: "0\t",
		},
		{
			in:       `""`,
			valid:    `""`,
			trailing: `""`,
		},
		{
			in:       "60",
			valid:    "60",
			trailing: "60",
		},
		{
			in:       "/[",
			valid:    "/[",
			trailing: "/[",
		},
		{
			in:       "0",
			valid:    "0",
			trailing: "0",
		},
	}

	for _, row := range table {
		for _, trailing := range []bool{true, false} {
			row := row
			trailing := trailing
			t.Run(fmt.Sprintf("row %#q", row.in), func(t *testing.T) {
				t.Parallel()
				var logs bytes.Buffer

				config := &jsoncomma.Config{
					Trailing: trailing,
					Logs:     &logs,
				}

				var actual bytes.Buffer
				actual.Grow(len(row.valid))

				written, err := jsoncomma.Fix(config, strings.NewReader(row.in), &actual)
				if err != nil {
					t.Errorf("in: %#q, err: %s", row.in, err)
				}
				fmt.Fprintf(&logs, "trailing: %t", trailing)
				t.Logf("logs\n%s", logs.String())

				var expected string
				if trailing {
					expected = row.trailing
				} else {
					expected = row.valid
				}

				actualString := actual.String()
				if int64(len(actualString)) != written {
					t.Errorf("in: %#q, output: %#q (%d bytes), yet written %d bytes", row.in, actualString, len(actualString), written)
				}
				if actualString != expected {
					t.Errorf("in: %#q\nactual:   %#q\nexpected: %#q", row.in, actualString, expected)
				}

			})
		}
	}
}

func BenchmarkFix(b *testing.B) {
	b.ReportAllocs()
	f, err := os.Open("../testdata/random.json")

	if err != nil {
		b.Fatalf("opening large JSON file: %s", err)
	}
	for i := 0; i < b.N; i++ {
		jsoncomma.Fix(&jsoncomma.Config{
			Trailing: false,
			Logs:     ioutil.Discard,
		}, f, ioutil.Discard)
	}
}

// ideally, Fix is as fast as io.Copy
func BenchmarkRef(b *testing.B) {
	b.ReportAllocs()
	f, err := os.Open("../testdata/random.json")

	if err != nil {
		b.Fatalf("opening large JSON file: %s", err)
	}

	for i := 0; i < b.N; i++ {
		io.Copy(ioutil.Discard, f)
	}
}
