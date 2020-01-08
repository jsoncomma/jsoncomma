// +build gofuzz

package jsoncomma

import (
	"bytes"
	"io/ioutil"
)

func FuzzLength(data []byte) int {
	written, err := Fix(&Config{}, bytes.NewReader(data), ioutil.Discard)
	if err != nil {
		// input generates an error, that means it's invalid. Not intersting.
		return 0
	}
	if written < len(data) {
		// written >= len(data) at all times (all we do is insert commas where needed)
		panic("Wrote less bytes than given.")
	}
	return 1
}
