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

func TestNoDiffMode(t *testing.T) {
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
		{
			in: `// hello world [1 2, ]
			[true false,]`,
			out: `// hello world [1 2, ]
			[true, false]`,
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
				Logs:     &logs,
				DiffMode: false,
			}

			var actual bytes.Buffer
			actual.Grow(len(row.out))

			read, written, err := jsoncomma.Fix(config, strings.NewReader(row.in), &actual)
			if err != nil {
				t.Errorf("in: %#q, err: %s", row.in, err)
			}
			t.Logf("logs\n%s", logs.String())

			actualString := actual.String()
			if int64(len(row.in)) != read {
				t.Errorf("in: %#q (%d bytes), yet read %d bytes", row.in, len(row.in), read)
			}
			if int64(len(actualString)) != written {
				t.Errorf("in: %#q, output: %#q (%d bytes), yet written %d bytes", row.in, actualString, len(actualString), written)
			}
			if actualString != row.out {
				t.Errorf("in: %#q\nactual:   %#q\nexpected: %#q", row.in, actualString, row.out)
			}
		})
	}
}

func TestDiffMode(t *testing.T) {
	input, err := ioutil.ReadFile("testdata/diffmode/input.notjson")
	if err != nil {
		t.Fatal(err)
	}
	expectedBytes, err := ioutil.ReadFile("testdata/diffmode/diff.txt")
	if err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	var actual bytes.Buffer

	read, written, err := jsoncomma.Fix(&jsoncomma.Config{
		Logs:     &logs,
		DiffMode: true,
	}, bytes.NewReader(input), &actual)

	t.Log(logs.String())
	if err != nil {
		t.Errorf("fixing: %s", err)
	}
	t.Logf("read %d bytes", read)
	t.Logf("wrote %d bytes", written)
	if read != int64(len(input)) {
		t.Errorf("bytes read (%d) != len(input) (%d)", read, len(input))
	}
	if written != int64(actual.Len()) {
		t.Errorf("bytes written (%d) != len(output) (%d)", written, actual.Len())
	}

	actualBytes := actual.Bytes()

	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Errorf("dismatch:\nactual:   %q\nexpected: %q", actualBytes, expectedBytes)
	}
}

func BenchmarkFix(b *testing.B) {
	b.ReportAllocs()
	f, err := os.Open("testdata/large.json")

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
	f, err := os.Open("testdata/large.json")

	if err != nil {
		b.Fatalf("opening large JSON file: %s", err)
	}

	for i := 0; i < b.N; i++ {
		io.Copy(ioutil.Discard, f)
	}
}
