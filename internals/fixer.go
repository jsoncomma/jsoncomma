package jsoncomma

import (
	"bufio"
	"log"
	"sync"
)

// all the unintersting stuff

// when to add a comma
// between an ending char (bool, end quote, digit, ], } and a ) and a starting char (start quote, digit, bool, [, {)

type Fixer struct {
	// TODO: config shouldn't be changed it has been given to fixer.
	// So, should we take a copy and not a reference?
	config *Config
	in     *bufio.Reader
	out    *bufio.Writer

	// the number of bytes written
	written int64
	// the number of bytes read
	read    int64
	last    byte

	// the number of commas added/removed. Hence it can be negative
	commasWritten int64

	log *log.Logger
}

func (f *Fixer) WriteByte(b byte) error {
	f.log.Printf("write %q", b)
	err := f.out.WriteByte(b)
	if err != nil {
		return err
	}
	f.written += 1
	return nil
}

func (f *Fixer) Write(b []byte) error {
	f.log.Printf("write %d", b)
	n, err := f.out.Write(b)
	f.written += int64(n)
	if err != nil {
		return err
	}
	return nil
}

func (f *Fixer) ReadByte() (byte, error) {
	b, err := f.in.ReadByte()
	f.log.Printf("read %q (%s)", b, err)
	if err != nil {
		return 0, err
	}
	f.read++
	return b, nil
}

func (f *Fixer) UnreadByte() error {
	err := f.in.UnreadByte()
	f.log.Printf("unread (%s)", err)
	if err != nil {
		return err
	}
	f.read--
	return nil
}

func (f *Fixer) ReadBytes(delim byte) ([]byte, error) {
	b, err := f.in.ReadBytes(delim)
	f.log.Printf("read %q (%d) (%s)", b, len(b), err)
	f.read += int64(len(b))
	if err != nil {
		return b, err
	}
	return b, nil
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

// written returns the number of bytes written
func (f *Fixer) Written() int64 {
	return f.written
}

// Read returns the number of bytes read
func (f *Fixer) Read() int64 {
	return f.read
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
