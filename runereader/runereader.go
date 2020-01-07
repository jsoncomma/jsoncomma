package runereader

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"
)

type RuneReader struct {
	*bufio.Reader
}

func NewRuneReader(r io.Reader) *RuneReader {
	// hum... really, all I care about is the peek method
	if bufr, ok := r.(*bufio.Reader); ok {
		return &RuneReader{bufr}
	}
	return &RuneReader{bufio.NewReader(r)}
}

// PeekRunes peeks at the next n runes. If an error occurs, then []runes
// will contain all the runes that were succesfully loaded before the error
// occured
func (r *RuneReader) PeekRunes(n int) ([]rune, error) {
	if n < 0 {
		return nil, bufio.ErrNegativeCount
	}

	bytesPeeked := 0
	runes := make([]rune, n)
	for i := 0; i < n; i++ {
		bytes, err := r.Peek(bytesPeeked + utf8.UTFMax)
		if err != nil && err != io.EOF {
			return runes[:i], err
		}

		var size int
		runes[i], size = utf8.DecodeRune(bytes[bytesPeeked:])
		bytesPeeked += size
		if err == io.EOF {
			return runes[:i+1], err
		}
	}

	fmt.Println("debugging:", runes)

	return runes, nil
}
