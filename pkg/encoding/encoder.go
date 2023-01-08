package encoding

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Encoder encodes runtime/trace events to a writer.
type Encoder struct {
	w             io.Writer    // output writer
	err           error        // sticky error
	buf           bytes.Buffer // scratch buf
	headerWritten bool         // true if header has been written
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes ev to the encoder's writer or returns an error.
func (e *Encoder) Encode(ev *Event) error {
	// Return error if any previous call to Encode failed
	if e.err != nil {
		return e.err
	}

	// Write header if not already done
	if !e.headerWritten {
		if _, e.err = e.w.Write(header); e.err != nil {
			return e.err
		}
		e.headerWritten = true
	}

	// Write event type and argument count
	narg := byte(len(ev.Args)) - 1
	if narg > 3 {
		narg = 3
	}
	if _, e.err = e.w.Write([]byte{byte(ev.Type) | narg<<6}); e.err != nil {
		return e.err
	}

	// Write string event
	if ev.Type == EventString {
		// Write string id
		if e.err = writeVarint(e.w, ev.Args[0]); e.err != nil {
			return e.err
		}
		// Write string length
		if e.err = writeVarint(e.w, uint64(len(ev.Str))); e.err != nil {
			return e.err
		}
		// Write string
		if _, e.err = e.w.Write(ev.Str); e.err != nil {
			return e.err
		}
		return nil
	} else if narg < 3 {
		// Write inlined arguments
		for _, arg := range ev.Args {
			if e.err = writeVarint(e.w, arg); e.err != nil {
				return e.err
			}
		}
	} else {
		// Write the arguments to e.buf to determine their encoded length
		e.buf.Reset()
		for _, arg := range ev.Args {
			if e.err = writeVarint(&e.buf, arg); e.err != nil {
				return e.err
			}
		}
		if ev.Type == EventStack {
			// Write the length of the encoded arguments to the e.w
			// Use writePaddedVarint to produce the same output as encoding/trace does which simplifies testing.
			if e.err = writePaddedVarint(e.w, uint64(e.buf.Len())); e.err != nil {
				return e.err
			}
		} else {
			// Write the length of the encoded arguments to the e.w
			if e.err = writeVarint(e.w, uint64(e.buf.Len())); e.err != nil {
				return e.err
			}
		}
		// Write the encoded arguments to e.w
		if _, e.err = e.w.Write(e.buf.Bytes()); e.err != nil {
			return e.err
		}
	}

	// Write user log event
	if ev.Type == EventUserLog {
		// Write string length
		if e.err = writeVarint(e.w, uint64(len(ev.Str))); e.err != nil {
			return e.err
		}
		// Write string
		if _, e.err = e.w.Write(ev.Str); e.err != nil {
			return e.err
		}
	}

	return nil
}

// writeVarint writes v as a varint to w or returns an error.
func writeVarint(w io.Writer, v uint64) error {
	var buf [10]byte
	n := binary.PutUvarint(buf[:], v)
	_, err := w.Write(buf[:n])
	return err
}

// writePaddedVarint writes v as a varint to w or returns an error.
// The varint is padded with 0x80 bytes to 10 bytes.
// This is done to produce exactly the same output as encoding/trace does which simplifies testing.
func writePaddedVarint(w io.Writer, v uint64) error {
	var buf [10]byte
	for i := 0; i < 10; i++ {
		if i < 10-1 {
			buf[i] = 0x80 | byte(v)
		} else {
			buf[i] = byte(v)
		}
		v >>= 7
	}
	_, err := w.Write(buf[:])
	return err
}
