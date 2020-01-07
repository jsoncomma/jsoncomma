package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"unicode"
)

type Config struct {
	Trailling bool
}

// when to add a comma
// between an ending char (bool, end quote, digit, ], } and a ) and a starting char (start quote, digit, bool, [, {)
// if trailling, between ending char and ending char

type Fixer struct {
	config *Config
	in     *bufio.Reader
	out    *bufio.Writer

	n               int
	lastSignificant byte
}

func (f *Fixer) WriteByte(b byte) error {
	err := f.out.WriteByte(b)
	if err != nil {
		return err
	}
	f.n += 1
	return nil
}

func (f *Fixer) Write(b []byte) error {
	n, err := f.out.Write(b)
	if err != nil {
		return err
	}
	f.n += n
	return nil
}

// consumeString reads the entire string and writes it to out, untouched.
// It handles backslashes (\")
func (f *Fixer) consumeString() error {
	var bytes []byte
	var err error

	for len(bytes) == 0 || bytes[len(bytes)-2] == '\\' {
		bytes, err = f.in.ReadBytes('"')
		if err := f.Write(bytes); err != nil {
			return err
		}
		if err != nil {
			return err
		}

		assert(len(bytes) >= 2, "len bytes should be greater than 2, got %d in %q", len(bytes), bytes)
	}
	if err := f.insertComma('"'); err != nil {
		return err
	}

	return nil
}

func (f *Fixer) consumeComment() error {
	var bytes []byte
	var err error

	// FIXME: better handling of different line endings
	bytes, err = f.in.ReadBytes('\n')
	if err := f.Write(bytes); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil

}

func isStart(b byte) bool {
	// f for false, t for true, n for null
	return b == '"' || b == '{' || b == '[' || (b >= '0' && b <= '9') || b == 'f' || b == 't' || b == 'n'
}

func isEnd(b byte) bool {
	// this is a bit dodgy. the 'e' is for false and true, the 'l' is for null
	return b == '"' || b == '}' || b == ']' || (b >= '0' && b <= '9') || b == 'e' || b == 'l'
}

func (f *Fixer) insertComma(lastSignificant byte) error {
	if !isEnd(lastSignificant) {
		return nil
	}

	// we can't peek, because we can only peek n bytes where n < buffer size
	// so if someone has a really long comment, then this will break
	// so, we read into a buffer *without writting to f.out*, and if we detect
	// that we need to insert a comma, then we f.out.Write and *then* f.out.Write
	// the buffer

	var bytesRead bytes.Buffer
	var next byte = ' '
	// here, we have to ignore spaces and comments, in a loop because you can
	// have whitespace, comment, whitespace, comment, etc...
	for {
		for {
			b, err := f.in.ReadByte()
			if err != nil {
				return err
			}
			next = b
			if !unicode.IsSpace(rune(next)) {
				if err := f.in.UnreadByte(); err != nil {
					return fmt.Errorf("unreading byte: %s", err)
				}
				break
			}
			if err := bytesRead.WriteByte(b); err != nil {
				return fmt.Errorf("writting to internal buffer: %s", err)
			}
		}
		if next != '/' {
			break
		}
		peek, err := f.in.Peek(1)
		if err != nil {
			return err
		}
		if peek[0] != '/' {
			// we don't actually have a comment
			break
		}

		// consume the comment
		// we can't use consume comment, because it writes to the buffer
		bytes, err := f.in.ReadBytes('\n')
		if err != nil {
			return err
		}
		written, err := bytesRead.Write(bytes)
		if err != nil {
			return fmt.Errorf("writing to internal buffer: %s", err)
		}
		if written != len(bytes) {
			return fmt.Errorf("writing to internal buffer: wrote %d bytes, expected %d", written, len(bytes))
		}

	}

	if isStart(next) {
		f.WriteByte(',')
	}

	shouldWrite := int64(bytesRead.Len())
	written, err := f.out.ReadFrom(&bytesRead)
	if err != nil {
		return fmt.Errorf("writing from internal buffer: %s", err)
	}
	if written != shouldWrite {
		return fmt.Errorf("writing from internal buffer: wrote %d bytes, expected %d", written, shouldWrite)
	}

	return nil
}

func (f *Fixer) Fix() error {

	for {
		// pay attention to
		// after a comment, consume the whole thing
		// within a string, consume the whole thing

		byte, err := f.in.ReadByte()
		if err := f.WriteByte(byte); err != nil {
			return err
		}
		if err != nil {
			return err
		}

		if byte == '"' {
			if err := f.consumeString(); err != nil {
				return err
			}
		} else if byte == '/' {
			next, err := f.in.Peek(1)
			if err != nil {
				return err
			}

			if next[0] == '/' {
				if err := f.consumeComment(); err != nil {
					return err
				}
			}

			// otherwise, don't do anything. We just peeked at the next
			// character, it's going to be consumed automatically
			// by something else
		} else {
			if err := f.insertComma(byte); err != nil {
				return err
			}
		}

	}
}

func (f *Fixer) Written() int {
	return f.n
}

func (f *Fixer) Flush() error {
	return f.out.Flush()
}

// Fix writes everything from in to out, just adding commas where needed
// returns the number of bytes written, and error
func Fix(config *Config, in io.Reader, out io.Writer) (int, error) {
	f := &Fixer{
		config: config,
		in:     bufio.NewReader(in),
		out:    bufio.NewWriter(out),
	}
	err := f.Fix()
	if err == io.EOF {
		err = nil
	}
	n := f.Written()
	if err != nil {
		return n, err
	}
	return n, f.Flush()
}
