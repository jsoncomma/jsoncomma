package jsoncomma_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	jsoncomma "github.com/jsoncomma/jsoncomma/internals"
)

func TestAddCommas(t *testing.T) {
	t.Parallel()
	table := []struct {
		in string
		// the expected output that is pure out JSON
		out string
	}{
		{
			in:  `{ "hello": "world" "oops": "world", }`,
			out: `{ "hello": "world", "oops": "world" }`,
		},
		{
			in:  `{ "hello": 2 "oops": "test" }`,
			out: `{ "hello": 2, "oops": "test" }`,
		},
		{
			in:  "{ \"hello\": 2\n\"oops\": \"test\", }",
			out: "{ \"hello\": 2,\n\"oops\": \"test\" }",
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
			["still" "works"],
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
				"weird":"whitespace, and sneaky [1 2]",}
			["still" "works"],
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
			in:  `{"hello": "world" "oops": "world",}`,
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
		{
			in:  `{"a": "b""c": "d",}`,
			out: `{"a": "b","c": "d"}`,
		},
		{
			in:  `[true true 123 false true,]`,
			out: `[true, true, 123, false, true]`,
		},
		{
			in:  `[1 2 3,4,5,6,7, [2, 3, 4],]`,
			out: `[1, 2, 3,4,5,6,7, [2, 3, 4]]`,
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
			in:  "60",
			out: "60",
		},
		{
			in:  "/[",
			out: "/[",
		},
		{
			in:  "0",
			out: "0",
		},
	}

	for i, row := range table {
		if i != 0 {
			continue
		}
		row := row
		t.Run(fmt.Sprintf("row %#q", row.in), func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer

			config := &jsoncomma.Config{
				Logs: &logs,
			}

			var actual bytes.Buffer
			actual.Grow(len(row.out))

			written, err := jsoncomma.Fix(config, strings.NewReader(row.in), &actual)
			if err != nil {
				t.Errorf("in: %#q, err: %s", row.in, err)
			}
			t.Logf("logs\n%s", logs.String())

			actualString := actual.String()
			if int64(len(actualString)) != written {
				t.Errorf("in: %#q, output: %#q (%d bytes), yet written %d bytes", row.in, actualString, len(actualString), written)
			}
			if actualString != row.out {
				t.Errorf("in: %#q\nactual:   %#q\nexpected: %#q", row.in, actualString, row.out)
			}
		})
	}
}

func BenchmarkFix(b *testing.B) {
	b.ReportAllocs()
	f, err := os.Open("../testdata/random.json")

	if err != nil {
		b.Fatalf("opening large JSON file: %s", err)
	}
	for i := 0; i < b.N; i++ {
		jsoncomma.Fix(&jsoncomma.Config{}, f, ioutil.Discard)
	}
}

// ideally, Fix is as fast as io.Copy. So, that's our reference
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
