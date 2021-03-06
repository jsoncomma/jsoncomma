package jsoncomma

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"
	"unicode"
)

type Config struct {
	Logs io.Writer
}

// when to add a comma
// between an ending char (bool, end quote, digit, ], } and a ) and a starting char (start quote, digit, bool, [, {)

type Fixer struct {
	config *Config
	in     *bufio.Reader
	out    *bufio.Writer

	n    int64
	last byte

	log *log.Logger
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
	f.n += int64(n)
	return nil
}

// consumeString reads the entire string and writes it to out, untouched.
// It handles backslashes (\")
func (f *Fixer) consumeString() error {
	var bytes []byte
	var err error

	for {
		bytes, err = f.in.ReadBytes('"')
		if err := f.Write(bytes); err != nil {
			return err
		}
		if f.log != nil {
			f.log.Printf("consume string: %#q", bytes)
		}
		if err != nil {
			return err
		}
		backslashCount := 0
		for {
			if len(bytes) - backslashCount - 2 < 0 || bytes[len(bytes) - backslashCount - 2] != '\\' {
				if f.log != nil {
					f.log.Printf("break")
				}
				break
			}
			backslashCount++
		}
		// if it's even, then that means that the quote isn't escaped
		if backslashCount % 2 == 0 {
			if f.log != nil {
				f.log.Printf("break with: %#q (count: %d)", bytes, backslashCount)
			}
			break
		}
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
	if f.log != nil {
		f.log.Printf("consume comment: %#q", bytes)
	}
	if err != nil {
		return err
	}
	return nil

}

func isPotentialStart(b byte) bool {
	// f for false, t for true, n for null
	return isStartPunctuation(b) || (b >= '0' && b <= '9') || b == 'f' || b == 't' || b == 'n'
}

func isStartPunctuation(b byte) bool {
	// f for false, t for true, n for null
	return b == '"' || b == '{' || b == '['
}

func isPotentialEnd(b byte) bool {
	return isEndPunctuation(b) || (b >= '0' && b <= '9') || b == 'e' || b == 'l'
}

func isEndPunctuation(b byte) bool {
	// this is a bit dodgy. the 'e' is for false and true, the 'l' is for null
	return b == '"' || b == '}' || b == ']'
}

func (f *Fixer) insertComma(last byte) (returnerr error) {
	// last is the last non-whitespace byte
	if !isPotentialEnd(last) {
		return nil
	}

	// we can't peek, because we can only peek n bytes where n < buffer size
	// so if someone has a really long comment, then this will break
	// so, we read into a buffer *without writting to f.out*, and if we detect
	// that we need to insert a comma, then we f.out.Write and *then* f.out.Write
	// the buffer

	var bytesRead bytes.Buffer
	var next byte = ' '

	// make sure we always write whatever we read to f.out
	defer func() {
		// if we encounter an error in this block, then it overwrites the
		// error returned (set returnerr)

		shouldWrite := int64(bytesRead.Len())
		written, err := f.out.ReadFrom(&bytesRead)
		if err != nil {
			if f.log != nil {
				f.log.Printf("overwriting error: %s", err)
			}
			returnerr = fmt.Errorf("writing from internal buffer: %s", err)
		}
		if written != shouldWrite {
			if f.log != nil {
				f.log.Printf("overwriting error: %s", err)
			}
			returnerr = fmt.Errorf("writing from internal buffer: wrote %d bytes, expected %d", written, shouldWrite)
		}
		f.n += written
	}()

	// here, we have to ignore spaces and comments, in a loop because you can
	// have whitespace, comment, whitespace, comment, etc...

	// we also need to make sure we have at least one space between
	// the last and the nextSignificant (otherwise 60 would
	// result in 6,0)
	spacesFound := 0
	for {

		// the loop here consumes all the spaces
		for {
			b, err := f.in.ReadByte()
			if err != nil {
				return err
			}
			next = b

			if !unicode.IsSpace(rune(next)) && next != ',' {
				if err := f.in.UnreadByte(); err != nil {
					return fmt.Errorf("unreading byte: %s", err)
				}
				break
			}
			if f.log != nil {
				f.log.Printf("space read/comma: '%q'", []byte{next})
			}
			if next != ',' {
				if err := bytesRead.WriteByte(next); err != nil {
					return fmt.Errorf("writting to internal buffer: %s", err)
				}
			}
			spacesFound += 1
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
		bytes, readerr := f.in.ReadBytes('\n')

		// make sure we write all the bytes we read, even if there is an error
		written, writeerr := bytesRead.Write(bytes)
		if writeerr != nil {
			return fmt.Errorf("writing to internal buffer: %s", writeerr)
		}
		if readerr != nil {
			return readerr
		}
		if written != len(bytes) {
			return fmt.Errorf("writing to internal buffer: wrote %d bytes, expected %d", written, len(bytes))
		}

	}

	// this is the magic. We insert a comma if any of those conditions are fulfilled

	// - we are between an end punctuation and a some potential start
	//     eg ...lue1""val... (last = " and next = ")
	//     eg ...lue1"true (last = " and next = t)
	addComma := isEndPunctuation(last) && isPotentialStart(next)

	// - we are between a potential end and a potential start AND THERE IS AT LEAST A SPACE
	//     eg 123 456 (last = 3 and next = 4).
	//     we need the space because otherwise 123 would be splited into 1,2,3
	addComma = addComma || (isPotentialEnd(last) && spacesFound >= 1 && isPotentialStart(next))

	if addComma {
		f.WriteByte(',')
	}

	return nil
}

func (f *Fixer) Fix() error {

	var prev, b byte
	var err error
	for {
		prev = b
		b, err = f.in.ReadByte()
		if err != nil {
			return err
		}
		if f.log != nil {
			f.log.Printf("regular read: '%q'", []byte{b})
		}
		if b != ',' {
			if err := f.WriteByte(b); err != nil {
				return err
			}
		}

		if b == '"' {
			if err := f.consumeString(); err != nil {
				return err
			}
		} else if b == '/' {
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
			if b == ',' {
				b = prev
			}
			if err := f.insertComma(b); err != nil {
				return err
			}
		}

	}
}

func (f *Fixer) Written() int64 {
	return f.n
}

func (f *Fixer) Flush() error {
	return f.out.Flush()
}

var readersPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReader(nil)
	},
}
var writersPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewWriter(nil)
	},
}

// Fix writes everything from in to out, just adding commas where needed
// returns the number of bytes written, and error
func Fix(config *Config, in io.Reader, out io.Writer) (int64, error) {

	var logger *log.Logger

	// this is much more efficient (prevents formating, where ioutil.Discard does the formating and then discards it)
	if config.Logs == ioutil.Discard {
		config.Logs = nil
	}

	if config.Logs != nil {
		logger = log.New(config.Logs, "", log.LstdFlags)
	}

	var bufin *bufio.Reader
	var bufout *bufio.Writer

	bufin = readersPool.Get().(*bufio.Reader)
	bufin.Reset(in)
	bufout = writersPool.Get().(*bufio.Writer)
	bufout.Reset(out)

	defer func() {
		readersPool.Put(bufin)
		writersPool.Put(bufout)
	}()

	f := &Fixer{
		config: config,
		in:     bufin,
		out:    bufout,

		log: logger,
	}

	if f.log != nil {
		f.log.Printf("config: %#v", config)
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
