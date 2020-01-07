package runereader

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPeekRune(t *testing.T) {
	var samples = []string{
		"Quick fox 世 jumped 界",
		// "யாமறிந்த மொழிகளிலே தமிழ்மொழி போல் இனிதாவது எங்கும் காணோம்",
		// " ಬಾ ಇಲ್ಲಿ ಸಂಭವಿಸು ಇಂದೆನ್ನ ಹೃದಯದಲಿ ",
		// "Sîne klâwen durh die wolken sint geslagen",
		// todo: fuzz!!!
	}

	const bufsize = 4096

	for _, string := range samples {
		reader := NewReader(bufio.NewReaderSize(strings.NewReader(string), bufsize))
		allRunes := []rune(string)

		assert(len(allRunes) == utf8.RuneCountInString(string), "I don't understand strings in go")

		// try to do all different peek sizes as possible
		for i := 0; i < utf8.RuneCountInString(string); i++ {
			// i is the number of runes to read

			runes, err := reader.PeekRunes(i)
			// i+1 because peek includes the ith rune
			if bytesInRunes(allRunes[:i+1]) > bufsize {
				if err != bufio.ErrBufferFull {
					t.Errorf("in %q, buffer size %d, peeking %d runes (%d bytes), expected bufio.ErrBufferFull, got %s", string, reader.Buffered(), i, bytesInRunes(allRunes[:i+1]), err)
				}
				if !runesEqual(runes, allRunes[:len(runes)]) {
					t.Errorf("in %q, peeking %d runes, expected %q, got %q", string, i, allRunes[:len(runes)], runes)
				}
				continue
			}

			if err != nil {
				t.Errorf("err peeking %d runes from %q: %s", i, string, err)
			}

			if !runesEqual(runes, allRunes[:i]) {
				t.Errorf("peek %d runes from %q. actual: %q, expected: %q", i, string, runes, allRunes[:i])
			}
		}

		// test behaviour around EOF

		for i := 0; i < 5; i++ {

			runes, err := reader.PeekRunes(len(allRunes) + i)

			if bytesInRunes(allRunes)+i > bufsize {
				// ErrBufferFull should take precedence of io.EOF?
				if err != io.EOF {
					t.Errorf("peeking %d runes after eof, expected io.EOF, got %s", i, err)
				}
				if !runesEqual(runes, allRunes) {
					t.Errorf("peeking %d runes after eof, expected entire runes %q, got %q", i, allRunes, runes)
				}

				continue
			}

			if err != io.EOF {
				t.Errorf("expected EOF after peeking %d runes after EOF from %q, got %s", i, string, err)
			}

			if !runesEqual(runes, allRunes) {
				t.Errorf("peeking %q %d runes after EOF, runes should be entire buffer %q, got %q", string, i, allRunes, runes)
			}
		}

	}

}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func bytesInRunes(runes []rune) int {
	sum := 0
	for _, r := range runes {
		sum += utf8.RuneLen(r)
	}
	return sum
}

func assert(c bool, format string, args ...interface{}) {
	if !c {
		fmt.Printf("[assertion error] "+format+"\n", args...)
		panic("assertion error")
	}
}
