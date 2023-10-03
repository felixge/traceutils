package encoding

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Decoder decodes runtime/trace events from a reader.
type Decoder struct {
	in         *reader
	br         bytes.Reader
	readHeader bool
	args       []byte // scratch buf
	version    int
}

// NewDecoder returns a new decoder that reads from r.
// Only supporting go 1.19 traces for now.
func NewDecoder(r io.Reader) *Decoder {
	p := &Decoder{in: newReader(r)}
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
	var buf [16]byte
	_, err := io.ReadFull(d.in, buf[:])
	if err != nil {
		return err
	} else if version, err := parseHeader(buf[:]); err != nil {
		return err
	} else {
		d.version = version
	}
	switch d.version {
	case 1019, 1021:
		break
	default:
		return fmt.Errorf("unsupported trace file version %v.%v %v", d.version/1000, d.version%1000, d.version)
	}
	return nil
}

// parseHeader parses trace header of the form "go 1.7 trace\x00\x00\x00\x00"
// and returns parsed version as 1007.
//
// This is copied from src/internal/trace/parser.go. in the Go source tree.
func parseHeader(buf []byte) (int, error) {
	if len(buf) != 16 {
		return 0, fmt.Errorf("bad header length")
	}
	if buf[0] != 'g' || buf[1] != 'o' || buf[2] != ' ' ||
		buf[3] < '1' || buf[3] > '9' ||
		buf[4] != '.' ||
		buf[5] < '1' || buf[5] > '9' {
		return 0, fmt.Errorf("not a trace file")
	}
	ver := int(buf[5] - '0')
	i := 0
	for ; buf[6+i] >= '0' && buf[6+i] <= '9' && i < 2; i++ {
		ver = ver*10 + int(buf[6+i]-'0')
	}
	ver += int(buf[3]-'0') * 1000
	if !bytes.Equal(buf[6+i:], []byte(" trace\x00\x00\x00\x00")[:10-i]) {
		return 0, fmt.Errorf("not a trace file")
	}
	return ver, nil
}

// Offset returns the current offset in the trace.
func (d *Decoder) Offset() int64 {
	return d.in.Offset
}

// Version returns the trace file version.
func (d *Decoder) Version() int {
	return d.version
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

		// Read length of argument byte slice
		length, err := readVal(d.in)
		if err != nil {
			return err
		}
		// Allocate argument byte slice
		for i := uint64(0); i < length; i++ {
			d.args = append(d.args, 0)
		}
		// Read argument byte slice
		_, err = io.ReadFull(d.in, d.args)
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

// readVal reads a base-128 varint encoded value from an io.Reader.
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

// newReader returns a new reader for r.
func newReader(r io.Reader) *reader {
	return &reader{r: bufio.NewReader(r)}
}

// reader is a buffered reader that keeps track of the number of bytes read.
type reader struct {
	r      *bufio.Reader
	Offset int64
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read or
// an error.
func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	r.Offset += int64(n)
	return n, err
}

// ReadByte reads a single byte from the reader.
func (r *reader) ReadByte() (byte, error) {
	b, err := r.r.ReadByte()
	if err == nil {
		r.Offset++
	}
	return b, err
}
