// +build gofuzz

package jsoncomma

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func FuzzLength(data []byte) int {
	var buf bytes.Buffer
	written, err := Fix(&Config{}, bytes.NewReader(data), &buf)
	if err != nil {
		panic(fmt.Sprintf("error fixing: %s", err))
	}
	output := buf.String()
	if int64(len(output)) != written {
		panic(fmt.Sprintf("output: %q (%d bytes), yet written = %d", output, len(output), written))
	}
	if written < int64(len(data)) {
		// written >= len(data) at all times (all we do is insert commas where needed)
		panic(fmt.Sprintf("Wrote less bytes than given (written = %d) (given %q, wrote %q)", written, data, output))
	}
	return 1
}

// FuzzJson parses data as some JSON, and makes sure after going through
// Fix, it's still the exact same string (if it's valid json, it shouldn't
// be touched unless Trailling == true, which we set to false)
func FuzzJson(data []byte) int {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		// unintersting data, it's not valid json
		return 0
	}
	var buf bytes.Buffer

	written, err := Fix(&Config{}, bytes.NewReader(data), &buf)
	if err != nil {
		panic(fmt.Sprintf("error fixing: %s", err))
	}

	actual := buf.Bytes()
	if written != int64(len(actual)) {
		panic(fmt.Sprintf("actual: %q (%d bytes), yet written = %d", actual, len(actual), written))
	}

	if !bytes.Equal(actual, data) {
		panic(fmt.Sprintf("dismatch:\noutput: %#q\nexpected: %#q", actual, data))
	}
	return 1
}
