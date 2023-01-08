package anon

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

// Bytes takes an argument s that is expected to contain a pkg.func or a file
// path and obfuscates it. The obfuscation is done by replacing all upper and
// lower case letters with "X" and "x" respectively. Additionally it keeps any
// ".go" suffix of s intact. For file paths ending in a valid package name, only
// the prefix of the path is obfuscated. For example:
func Bytes(s []byte, packages []string) {
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
	if bytes.HasSuffix(b, []byte(".go")) {
		b = b[:len(b)-3]
	}

	// iterate over all utf8 runes in b
	// if rune is not in allowed, replace with "x"
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
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
