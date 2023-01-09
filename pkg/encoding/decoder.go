package encoding

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Decoder decodes runtime/trace events from a reader.
type Decoder struct {
	in         *bufio.Reader
	br         bytes.Reader
	readHeader bool
	args       []byte // scratch buf
}

// NewDecoder returns a new decoder that reads from r.
// Only supporting go 1.19 traces for now.
func NewDecoder(r io.Reader) *Decoder {
	p := &Decoder{
		in:   bufio.NewReader(r),
		args: make([]byte, 0, 1<<10),
	}
	return p
}

// header is the trace file header.
var header = func() []byte {
	header := make([]byte, 16)
	// Only supporting go 1.19 traces for now
	// TODO: support older traces
	copy(header, "go 1.19 trace")
	return header
}()

// header reads the header and returns an error if it is invalid.
func (d *Decoder) header() error {
	// Read header
	buf := make([]byte, len(header))
	_, err := io.ReadFull(d.in, buf)
	if err != nil {
		return err
	} else if !bytes.Equal(buf, header) {
		// Fail if header is invalid
		return fmt.Errorf("invalid header: %q", string(buf))
	}
	return nil
}

// Decode parses an event or returns an error.
func (d *Decoder) Decode(e *Event) error {
	if !d.readHeader {
		if err := d.header(); err != nil {
			return err
		}
		d.readHeader = true
	}

	// Read event type and argument count contained in the first byte
	firstByte, err := d.in.ReadByte()
	if err != nil {
		return err
	}

	// Decode event type
	e.Type = EventType(firstByte & 0b00111111)
	// Reset event and decoder state
	e.Args = e.Args[:0]
	e.Str = e.Str[:0]
	d.args = d.args[:0]
	// Decode argument count
	narg := firstByte>>6 + 1

	// Read string event
	if e.Type == EventString {
		// Read string id and add it to e.Args
		id, err := readVal(d.in)
		if err != nil {
			return err
		}
		e.Args = append(e.Args, id)
		e.Str, err = readNext(d.in, e.Str)
		if err != nil {
			return err
		}
	} else if narg < 4 {
		// If the number of arguments is less than 4, the arguments directly
		// follow the first byte as base-128 varints.
		for i := 0; i < int(narg); i++ {
			// Read argument and add it to e.Args
			arg, err := readVal(d.in)
			if err != nil {
				return err
			}
			e.Args = append(e.Args, arg)
		}
	} else {
		// If the number of arguments is greater than 3, the arguments are
		// encoded as a base-128 varint length followed by a byte slice of
		// base-128 varints.

		d.args, err = readNext(d.in, d.args)
		if err != nil {
			return err
		}

		// Decode argument byte slice.
		// Reuse d.br for this to avoid allocations.
		d.br.Reset(d.args)
		// Read arguments and add them to e.Args
		for {
			// Read argument and add it to e.Args
			arg, err := readVal(&d.br)
			if err == io.EOF {
				// Stop reading arguments when we reach the end of the byte slice
				break
			} else if err != nil {
				return err
			}
			e.Args = append(e.Args, arg)
		}
	}

	// Read user log event. This is a special case because the string is
	// encoded as a base-128 varint length followed by a byte slice of bytes.
	if e.Type == EventUserLog {
		// Read string length
		length, err := readVal(d.in)
		if err != nil {
			return err
		}
		// Allocate e.Str of the correct length
		for i := uint64(0); i < length; i++ {
			e.Str = append(e.Str, 0)
		}
		// Read string into e.Str
		if _, err := io.ReadFull(d.in, e.Str); err != nil {
			return err
		}
	}
	return nil
}

func readNext(r *bufio.Reader, buf []byte) ([]byte, error) {
	// Read length of byte slice
	length, err := readVal(r)
	if err != nil {
		return nil, err
	}
	// Grow byte slice
	if len(buf) < int(length) {
		buf = append(buf, make([]byte, int(length)-len(buf))...)
	}
	// Read byte slice
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	return buf, err
}

// readVal reads a base-128 varint encoded value from r.
func readVal(r io.ByteReader) (uint64, error) {
	var val uint64 // decoded value
	var shift uint // number of bits to shift
	for {
		// Read byte
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		// Decode byte, shifting it left by the number of bits already decoded,
		// and add it to the decoded value.
		val |= uint64(b&0x7f) << shift
		// Stop reading bytes when the most significant bit is 0
		if b&0x80 == 0 {
			break
		}
		// Increment shift by 7 to shift the next byte left by 7 bits
		shift += 7
	}
	return val, nil
}
