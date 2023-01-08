package encoding

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type Decoder struct {
	in   *bufio.Reader
	br   bytes.Reader
	args []byte // scratch buf
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	p := &Decoder{in: bufio.NewReader(r)}
	return p, p.header()
}

var header = func() []byte {
	header := make([]byte, 16)
	// only supporting go 1.19 traces for now
	copy(header, "go 1.19 trace")
	return header
}()

// header reads the header and returns an error if it is invalid.
func (d *Decoder) header() error {
	buf := make([]byte, len(header))
	_, err := io.ReadFull(d.in, buf)
	if err != nil {
		return err
	} else if !bytes.Equal(buf, header) {
		return fmt.Errorf("invalid header: %q", string(buf))
	}
	return nil
}

// Decode parses an event or returns an error.
func (d *Decoder) Decode(e *Event) error {
	b, err := d.in.ReadByte()
	if err != nil {
		return err
	}

	e.Type = EventType(b & 0b00111111)
	e.Args = e.Args[:0]
	e.Str = e.Str[:0]
	d.args = d.args[:0]

	narg := b>>6 + 1
	if e.Type == EventString {
		id, err := readVal(d.in)
		if err != nil {
			return err
		}
		e.Args = append(e.Args, id)
		length, err := readVal(d.in)
		if err != nil {
			return err
		}
		for i := uint64(0); i < length; i++ {
			e.Str = append(e.Str, 0)
		}

		// read string into e.Str
		if _, err := io.ReadFull(d.in, e.Str); err != nil {
			return err
		}
	} else if narg < 4 {
		// inlined arguments
		for i := 0; i < int(narg); i++ {
			arg, err := readVal(d.in)
			if err != nil {
				return err
			}
			e.Args = append(e.Args, arg)
		}
	} else {
		length, err := readVal(d.in)
		if err != nil {
			return err
		}
		for i := uint64(0); i < length; i++ {
			d.args = append(d.args, 0)
		}
		_, err = io.ReadFull(d.in, d.args)
		if err != nil {
			return err
		}
		d.br.Reset(d.args)
		for {
			arg, err := readVal(&d.br)
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			e.Args = append(e.Args, arg)
		}
	}

	if e.Type == EventUserLog {
		length, err := readVal(d.in)
		if err != nil {
			return err
		}
		for i := uint64(0); i < length; i++ {
			e.Str = append(e.Str, 0)
		}
		if _, err := io.ReadFull(d.in, e.Str); err != nil {
			return err
		}

	}

	return nil
}

// readVal reads a base-128 varint encoded value from an io.Reader.
func readVal(r io.ByteReader) (uint64, error) {
	var val uint64
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		val |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return val, nil
}
