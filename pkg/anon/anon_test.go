package anon_test

import (
	"testing"

	"github.com/felixge/traceutils/pkg/anon"
)

func TestBytes(t *testing.T) {
	allowed := []string{"runtime", "encoding/json"}
	tests := []struct {
		name string
		s    []byte
		want string
	}{
		{
			name: "pkg.func: ok",
			s:    []byte("encoding/json.Marshal"),
			want: "encoding/json.Marshal",
		},

		{
			name: "pkg.func: wrong prefix",
			s:    []byte("my/encoding/json.Marshal"),
			want: "xx/xxxxxxxx/xxxx.Xxxxxxx",
		},

		{
			name: "pkg.func: wrong suffix",
			s:    []byte("encoding/json/foo.Marshal"),
			want: "xxxxxxxx/xxxx/xxx.Xxxxxxx",
		},

		{
			name: "path: ok",
			s:    []byte("/src/runtime/proc.go"),
			want: "/src/runtime/proc.go",
		},

		{
			name: "path: replace prefix",
			s:    []byte("/home/Bob/src/runtime/proc.go"),
			want: "/xxxx/Xxx/src/runtime/proc.go",
		},

		{
			name: "path: replace all",
			s:    []byte("/home/Bob/src/runtime/foo/proc.go"),
			want: "/xxxx/Xxx/xxx/xxxxxxx/xxx/xxxx.go",
		},

		{
			name: "path: all tricky",
			s:    []byte("/home/Bob/src/runtime"),
			want: "/xxxx/Xxx/xxx/xxxxxxx",
		},

		{
			name: "path: all tricky 2",
			s:    []byte("/home/Bob/src/runtime/"),
			want: "/xxxx/Xxx/xxx/xxxxxxx/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anon.Bytes(tt.s, allowed)
			if got := string(tt.s); got != tt.want {
				t.Errorf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

// pkgs, err := packages.Load(nil, "std")
// if err != nil {
// 	panic(err)
// }
// for i, p := range pkgs {
// 	fmt.Println(i, p)
// }
