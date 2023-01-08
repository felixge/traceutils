package anonymize

import (
	"bytes"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/felixge/traceutils/pkg/encoding"
	"golang.org/x/tools/go/packages"
)

// AnonymizeTrace read a runtime/trace file from r and writes an obfuscated
// version of it to w. The obfuscation is done by replacing all references to
// file paths and packages not found Go's standard library with obfuscated
// versions. The obfuscation is done by replacing all upper and lower case
// letters with "X" and "x" respectively. Additionally it keeps any ".go"
// suffixes intact. For file paths ending in a stdlib package name, only the
// prefix of the path is obfuscated. On success AnonymizeTrace returns nil. If
// an error occurs, AnonymizeTrace returns the error.
func AnonymizeTrace(r io.Reader, w io.Writer) error {
	// Determine stdlib packages
	pkgs, err := packages.Load(nil, "std")
	if err != nil {
		return err
	}
	var stdlibPkgs []string
	for _, pkg := range pkgs {
		stdlibPkgs = append(stdlibPkgs, pkg.PkgPath)
	}

	// Initialize encoder and decoder
	enc := encoding.NewEncoder(w)
	dec, err := encoding.NewDecoder(r)
	if err != nil {
		return err
	}
	// Obfuscate all string events
	for {
		var ev encoding.Event
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		anonymizeString(ev.Str, stdlibPkgs)

		if err := enc.Encode(&ev); err != nil {
			return err
		}
	}
}

// anonymizeString takes an argument s that is expected to contain a pkg.func or
// a file path and obfuscates it. The obfuscation is done by replacing all upper
// and lower case letters with "X" and "x" respectively. Additionally it keeps
// any ".go" suffix of s intact. For file paths ending in a valid package name,
// only the prefix of the path is obfuscated. For example:
func anonymizeString(s []byte, packages []string) {
	if len(s) == 0 {
		return
	}

	if s[0] != '/' {
		// s is probably a pkg.func
		pkg, _, found := bytes.Cut(s, []byte("."))
		if !found {
			obfuscate(s)
			return
		}
		for _, stdlibPkg := range packages {
			if bytes.Equal(pkg, []byte(stdlibPkg)) {
				return
			}
		}
		obfuscate(s)
		return
	}

	// s is probably a file path
	var longest struct {
		length int
		prefix []byte
		suffix []byte
	}
	for _, pkg := range packages {
		sep := append([]byte("src/"), []byte(pkg)...)
		prefix, suffix, found := bytes.Cut(s, sep)
		if found && len(pkg) > longest.length && len(suffix) > 1 && !bytes.Contains(suffix[1:], []byte("/")) {
			longest.length = len(pkg)
			longest.prefix = prefix
			longest.suffix = suffix
		}
	}
	if longest.length == 0 {
		obfuscate(s)
	} else {
		obfuscate(longest.prefix)
	}

}

// obfuscate replaces all upper and lower case letters with "X" and "x"
// respectively. Additionally it keeps any ".go" suffix of b intact.
func obfuscate(b []byte) {
	// Keep ".go" suffix intact
	if bytes.HasSuffix(b, []byte(".go")) {
		b = b[:len(b)-3]
	}

	// Iterate over all utf8 runes in b
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		// Replace all upper and lower case letters with "X" and "x"
		if unicode.IsUpper(r) {
			for j := 0; j < size; j++ {
				b[i+j] = 'X'
			}
		} else if unicode.IsLower(r) {
			for j := 0; j < size; j++ {
				b[i+j] = 'x'
			}
		}
		i += size
	}
}
