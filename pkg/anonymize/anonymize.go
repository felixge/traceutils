package anonymize

import (
	"bytes"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/felixge/traceutils/pkg/encoding"
	"golang.org/x/tools/go/packages"
)

var stdlibPkgs = func() []string {
	// Determine stdlib packages
	pkgs, err := packages.Load(nil, "std")
	if err != nil {
		panic(err)
	}
	var stdlibPkgs []string
	for _, pkg := range pkgs {
		stdlibPkgs = append(stdlibPkgs, pkg.PkgPath)
	}
	return stdlibPkgs
}()

// AnonymizeTrace read a runtime/trace file from r and writes an obfuscated
// version of it to w. The obfuscation is done by replacing all references to
// file paths and packages not found Go's standard library with obfuscated
// versions. The obfuscation is done by replacing all upper and lower case
// letters with "X" and "x" respectively. Additionally it keeps any ".go"
// suffixes intact. For file paths ending in a stdlib package name, only the
// prefix of the path is obfuscated. On success AnonymizeTrace returns nil. If
// an error occurs, AnonymizeTrace returns the error.
//
// TODO: This function is pretty slow, maybe we can do better? It takes about
// 6min17s to anonymize a 280MB trace on my machine.
func AnonymizeTrace(r io.Reader, w io.Writer) error {
	// Initialize encoder and decoder
	enc := encoding.NewEncoder(w)
	dec, err := encoding.NewDecoder(r)
	if err != nil {
		return err
	}
	// Obfuscate all string events
	for {
		// Decode event
		var ev encoding.Event
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				// We're done
				return nil
			}
			return err
		}

		// Obfuscate string
		// TODO: Doing this concurrently might be nice for bigger trace.
		anonymizeString(ev.Str, stdlibPkgs)

		// Encode the obfuscated event
		if err := enc.Encode(&ev); err != nil {
			return err
		}
	}
}

// gcMarkWorkerModeStrings is a map of strings emitted at the start of a
// runtime/trace that we don't want to obfuscate. This is copied from
// runtime/trace.go in the Go source tree.
var gcMarkWorkerModeStrings = map[string]bool{
	"Not worker":      true,
	"GC (dedicated)":  true,
	"GC (fractional)": true,
	"GC (idle)":       true,
}

// anonymizeString takes an argument s that is expected to contain a pkg.func or
// a file path and obfuscates it. The obfuscation is done by replacing all upper
// and lower case letters with "X" and "x" respectively. Additionally it keeps
// any ".go" suffix of s intact. For file paths ending in a valid package name,
// only the prefix of the path is obfuscated. For example:
// TODO: This function is kind of slow, maybe we can do better?
func anonymizeString(s []byte, packages []string) {
	if len(s) == 0 {
		return
	}

	// Don't obfuscate the gcMarkWorkerModeStrings found in every trace
	if _, ok := gcMarkWorkerModeStrings[string(s)]; ok {
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
