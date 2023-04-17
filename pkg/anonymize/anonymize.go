package anonymize

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/felixge/traceutils/pkg/encoding"
	"golang.org/x/tools/go/packages"
)

// replacement returns a string that is used to obfuscate file paths and
// packages not found in Go's standard library. It's a function to ensure the
// slice won't be overwritten by the caller.
var replacement = func() []byte { return []byte("XXX") }

// AnonymizeTrace reads a runtime/trace file from r and writes an obfuscated
// version of it to w. The obfuscation is done by replacing all references to
// file paths and packages not found Go's standard library with obfuscated
// versions. The obfuscation is done by replacing all letters that need
// obfuscation with "XXX". Additionally it keeps any ".go" suffixes and special
// GC strings intact. For file paths ending in a stdlib package name, only the
// prefix of the path is obfuscated. On success AnonymizeTrace returns nil. If
// an error occurs, AnonymizeTrace returns the
// error.
func AnonymizeTrace(r io.Reader, w io.Writer) error {
	// Initialize encoder and decoder
	buf := bufio.NewWriter(w)
	enc := encoding.NewEncoder(buf)
	dec := encoding.NewDecoder(r)

	// Obfuscate all string events
	var ev encoding.Event
	for {
		// Decode event
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				// We're done
				return buf.Flush()
			}
			return err
		}

		// Obfuscate string
		ev.Str = anonymizeString(ev.Str)

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

// anonymizeString obfuscates the given string and returns the obfuscated
// version.
func anonymizeString(s []byte) []byte {
	if len(s) == 0 {
		return s
	}

	// Don't obfuscate the gcMarkWorkerModeStrings found in every trace
	if _, ok := gcMarkWorkerModeStrings[string(s)]; ok {
		return s
	}

	if s[0] != '/' {
		// s is probably a pkg.func
		return anonymizeFunc(s)
	}
	// s is probably a file path
	return anonymizePath(s)
}

func anonymizeFunc(s []byte) []byte {
	// s is probably a pkg.func
	pkg, _, found := bytes.Cut(s, []byte("."))
	if !found {
		return replacement()
	}
	for _, stdlibPkg := range stdlibPkgs {
		if bytes.Equal(pkg, []byte(stdlibPkg)) {
			return s
		}
	}
	return replacement()
}

func anonymizePath(s []byte) []byte {
	var srcSep = []byte("/src/")
	head, tail := bytesLastSplit(s, srcSep)
	if head == nil || tail == nil {
		return replacement()
	}
	if pathInStdLib(string(tail)) {
		return append(append(replacement(), srcSep...), tail...)
	}
	if bytes.HasSuffix(s, []byte(".go")) {
		return append(replacement(), []byte(".go")...)
	}
	return replacement()
}

func bytesLastSplit(s, sep []byte) (head, tail []byte) {
	if len(sep) == 0 {
		return s, nil
	}
	if idx := bytes.LastIndex(s, sep); idx >= 0 {
		return s[:idx], s[idx+len(sep):]
	}
	return nil, s
}

var stdlibPkgs = func() [][]byte {
	// Determine stdlib packages
	pkgs, err := packages.Load(nil, "std")
	if err != nil {
		panic(err)
	}
	var stdlibPkgs [][]byte
	for _, pkg := range pkgs {
		stdlibPkgs = append(stdlibPkgs, []byte(pkg.PkgPath))
	}
	return stdlibPkgs
}()

// pathInStdLib returns true if the given path is in Go's standard
// library.
func pathInStdLib(path string) bool {
	srcPath := filepath.Join(runtime.GOROOT(), "src", path)
	stat, err := os.Stat(srcPath)
	return err == nil && !stat.IsDir()
}
