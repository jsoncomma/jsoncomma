// +build gofuzz

package jsoncomma

import (
	"bytes"
	"fmt"
)

func FuzzLength(data []byte) int {
	var buf bytes.Buffer
	written, err := Fix(&Config{}, bytes.NewReader(data), &buf)
	if err != nil {
		// input generates an error, that means it's invalid. Not intersting.
		return 0
	}
	output := buf.String()
	if len(output) != written {
		panic(fmt.Sprintf("output: %q (%d bytes), yet written = %d", output, len(output), written))
	}
	if written < len(data) {
		// written >= len(data) at all times (all we do is insert commas where needed)
		panic(fmt.Sprintf("Wrote less bytes than given (written = %d) (given %q, wrote %q)", written, data, output))
	}
	return 1
}
