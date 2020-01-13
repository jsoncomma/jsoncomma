package jsoncomma

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"unicode"
)

type Config struct {
	// Logs is the debug logger. If set to nil or ioutil.Discard,
	// no log operation will occur (very significant perfs improvement)
	Logs io.Writer

	// Instead of re-writing the whole file, just write a bunch of lines
	// format: {number} {+|-} where number is the offset (0 based), + means insert, - remove
	// eg
	// 10+
	// 15-
	// 20+
	// 26+
	// means insert at 10, then remove at 15, add at 20 and at 26.
	// It's important to process these instructions sequentially. 26 refers to the
	// position where the client should insert a comma after having inserted 2 commas
	// and removed one (do you see how everything would be shifted by one character?)
	// 89- means remove character after position 89 (between 89 and 90)
	DiffMode bool
}

// Fix writes everything from in to out, just adding commas where needed
// returns the number of bytes read, number of bytes written, and error
func Fix(config *Config, in io.Reader, out io.Writer) (int64, int64, error) {

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
	read := f.Read()
	written := f.Written()

	if err == io.EOF {
		err = nil
	}
	if err != nil {
		return read, written, err
	}
	return read, written, f.Flush()
}

func (f *Fixer) Fix() error {
	for {
		current, err := f.ReadByte()
		if err != nil {
			return err
		}

		if current == '/' {
			peek, err := f.in.Peek(1)
			if err != nil {
				return err
			}
			if peek[0] == '/' {
				_, err := f.ReadBytes('\n')
				if err != nil {
					return err
				}
				continue
			}
		}

		if current == '"' {
			for {
				buf, err := f.ReadBytes('"')
				if err != nil {
					return err
				}
				if len(buf) < 1 {
					panic("assertion error: should have at least the end quote in buf")
				}
				if len(buf) < 2 || buf[len(buf)-2] != '\\' {
					break
				}
			}

			// set current = '"' (it's already the case)
		}

		if !isPotentialEnd(current) {
			continue
		}

		if f.log != nil {
			f.log.Printf("potential end %q", current)
		}

		lastSignificant := current

		// the number of bytes read
		offset := f.read + f.commasWritten

		atLeastOneSpace := false
		isComma := false
		onlyHadSpaceOrComment := false
		peek := -1

		// consume every comment and whitespace to check if the next character is an end

		for {
			b, err := f.ReadByte()
			if err != nil {
				return err
			}

			if b == '/' {
				peek, err := f.in.Peek(1)
				if err != nil {
					return err
				}
				// consume the comment
				if peek[0] == '/' {
					atLeastOneSpace = true
					_, err := f.ReadBytes('\n')
					if err != nil {
						return err
					}
					// ignore the rest of the loop because every check is going to fail anyway,
					// b == '/'
					continue
				} else {
					// it's not a comment, this / is breaking (we only accept whitespace and comments)
					onlyHadSpaceOrComment = false
					break // here we also want to continue the outer loop
				}
			}
			if b == ',' {
				isComma = true
			} else if !unicode.IsSpace(rune(b)) {
				// we don't know what this byte is. We know it's not a space or a comment,
				// but we shouldn't consume it
				if err := f.UnreadByte(); err != nil {
					return err
				}
				// next is kind of like a peek.
				peek = int(b)
				break
			}

			atLeastOneSpace = true
			onlyHadSpaceOrComment = true

		}

		if !onlyHadSpaceOrComment {
			if f.log != nil {
				f.log.Printf("didn't only have space or comments, skip")
			}
			continue
		}

		if peek == -1 {
			panic("assertion error: next wasn't populated (-1)")
		}

		next := byte(peek)
		// we should only get here if the text between is lastSignificant and current is space/comments or nothing

		// this is the magic. We insert a comma if any of those conditions are fulfilled

		// - we are between an end punctuation and a some potential start
		//     eg ...lue1""val... (last = " and next = ")
		//     eg ...lue1"true (last = " and next = t)
		endStart := isEndPunctuation(lastSignificant) && isPotentialStart(next)

		// - we are between a potential end and a potential start AND THERE IS AT LEAST A SPACE
		//     eg 123 456 (last = 3 and next = 4).
		//     we need the space because otherwise 123 would be splited into 1,2,3
		endSpaceStart := (isPotentialEnd(lastSignificant) && atLeastOneSpace && isPotentialStart(next))

		addComma := endStart || endSpaceStart

		if isComma && !addComma {
			if f.log != nil {
				f.log.Printf("remove comma at %d (end-start: %t end-space-start: %t) last: %c next: %c", offset, endStart, endSpaceStart, lastSignificant, next)
			}
			written, err := fmt.Fprintf(f.out, "%d-\n", offset)
			if err != nil {
				return err
			}
			f.written += int64(written)
			f.commasWritten--
		} else if !isComma && addComma {
			if f.log != nil {
				f.log.Printf("add comma at %d (end-start: %t end-space-start: %t) last: %c next: %c", offset, endStart, endSpaceStart, lastSignificant, next)
			}
			written, err := fmt.Fprintf(f.out, "%d+\n", offset)
			if err != nil {
				return err
			}
			f.written += int64(written)
			f.commasWritten++
		} else {
			f.log.Printf("no change needed %d (isComma: %t addComma: %t) last: %c current: %c", offset, isComma, addComma, lastSignificant, current)
		}

	}
}
