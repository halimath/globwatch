package pattern

import (
	"errors"
	"testing"

	"github.com/halimath/fsmock"

	. "github.com/halimath/expect-go"
)

type test struct {
	pattern, f string
	match      bool
	err        error
}

var tests = []test{
	// Test cases not covered by path.Match
	{"main.go", "main.go", true, nil},
	{"main_test.go", "main_test.go", true, nil},
	{"foo/foo_test.go", "foo/foo_test.go", true, nil},
	{"?.go", "m.go", true, nil},
	{"*.go", "main.go", true, nil},
	{"**/*.go", "main.go", true, nil},
	{"*.go", "*.go", true, nil},

	{"//", "", false, ErrBadPattern},
	{"foo//", "", false, ErrBadPattern},
	{"*?.go", "", false, ErrBadPattern},
	{"?*.go", "", false, ErrBadPattern},
	{"**?.go", "", false, ErrBadPattern},
	{"**f", "", false, ErrBadPattern},
	{"[a-", "", false, ErrBadPattern},
	{"[a-\\", "", false, ErrBadPattern},
	{"[\\", "", false, ErrBadPattern},

	{"**/m.go", "foo.go", false, nil},
	{"**/m.go", "foo/a.go", false, nil},
	{"**/m.go", "m.go", true, nil},
	{"**/m.go", "foo/m.go", true, nil},
	{"**/m.go", "bar/m.go", true, nil},
	{"**/m.go", "foo/bar/m.go", true, nil},

	{"ab[cde]", "abc", true, nil},
	{"ab[cde]", "abd", true, nil},
	{"ab[cde]", "abe", true, nil},
	{"ab[+-\\-]", "ab-", true, nil},
	{"ab[\\--a]", "ab-", true, nil},

	{"[a-fA-F]", "a", true, nil},
	{"[a-fA-F]", "f", true, nil},
	{"[a-fA-F]", "A", true, nil},
	{"[a-fA-F]", "F", true, nil},

	// The following test cases are taken from
	// https://github.com/golang/go/blob/master/src/path/match_test.go and are
	// provided here to test compatebility of the match implementation with the
	// test cases from the golang standard lib.
	{"abc", "abc", true, nil},
	{"*", "abc", true, nil},
	{"*c", "abc", true, nil},
	{"a*", "a", true, nil},
	{"a*", "abc", true, nil},
	{"a*", "ab/c", false, nil},
	{"a*/b", "abc/b", true, nil},
	{"a*/b", "a/c/b", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
	{"ab[c]", "abc", true, nil},
	{"ab[b-d]", "abc", true, nil},
	{"ab[e-g]", "abc", false, nil},
	{"ab[^c]", "abc", false, nil},
	{"ab[^b-d]", "abc", false, nil},
	{"ab[^e-g]", "abc", true, nil},
	{"a\\*b", "a*b", true, nil},
	{"a\\*b", "ab", false, nil},
	{"a?b", "a☺b", true, nil},
	{"a[^a]b", "a☺b", true, nil},
	{"a???b", "a☺b", false, nil},
	{"a[^a][^a][^a]b", "a☺b", false, nil},
	{"[a-ζ]*", "α", true, nil},
	{"*[a-ζ]", "A", false, nil},
	{"a?b", "a/b", false, nil},
	{"a*b", "a/b", false, nil},
	{"[\\]a]", "]", true, nil},
	{"[\\-]", "-", true, nil},
	{"[x\\-]", "x", true, nil},
	{"[x\\-]", "-", true, nil},
	{"[x\\-]", "z", false, nil},
	{"[\\-x]", "x", true, nil},
	{"[\\-x]", "-", true, nil},
	{"[\\-x]", "a", false, nil},
	{"[]a]", "]", false, ErrBadPattern},
	{"[-]", "-", false, ErrBadPattern},
	{"[x-]", "x", false, ErrBadPattern},
	{"[x-]", "-", false, ErrBadPattern},
	{"[x-]", "z", false, ErrBadPattern},
	{"[-x]", "x", false, ErrBadPattern},
	{"[-x]", "-", false, ErrBadPattern},
	{"[-x]", "a", false, ErrBadPattern},
	{"\\", "a", false, ErrBadPattern},
	{"[a-b-c]", "a", false, ErrBadPattern},
	{"[", "a", false, ErrBadPattern},
	{"[^", "a", false, ErrBadPattern},
	{"[^bc", "a", false, ErrBadPattern},
	{"a[", "a", false, ErrBadPattern},
	{"a[", "ab", false, ErrBadPattern},
	{"a[", "x", false, ErrBadPattern},
	{"a/b[", "x", false, ErrBadPattern},
	{"*x", "xxx", true, nil},
}

func TestPattern_Match(t *testing.T) {
	for _, tt := range tests {
		pat, err := New(tt.pattern)
		if err != tt.err && !errors.Is(err, tt.err) {
			t.Errorf("New(%#q): wanted error %v but got %v", tt.pattern, tt.err, err)
		}

		if pat != nil {
			match := pat.Match(tt.f)
			if match != tt.match {
				t.Errorf("New(%#q).Match(%#q): wanted match %v but got %v", tt.pattern, tt.f, tt.match, match)
			}
		}
	}
}

func TestPattern_GlobFS(t *testing.T) {
	fsys := fsmock.New(fsmock.NewDir("",
		fsmock.EmptyFile("go.mod"),
		fsmock.EmptyFile("go.sum"),
		fsmock.NewDir("cmd",
			fsmock.EmptyFile("main.go"),
			fsmock.EmptyFile("main_test.go"),
		),
		fsmock.NewDir("internal",
			fsmock.NewDir("tool",
				fsmock.EmptyFile("tool.go"),
				fsmock.EmptyFile("tool_test.go"),
			),
			fsmock.NewDir("cli",
				fsmock.EmptyFile("cli.go"),
				fsmock.EmptyFile("cli_test.go"),
			),
		),
	))

	pat, err := New("**/*_test.go")
	if err != nil {
		t.Fatal(err)
	}

	files, err := pat.GlobFS(fsys, "")
	if err != nil {
		t.Fatal(err)
	}

	ExpectThat(t, files).Is(DeepEqual([]string{
		"cmd/main_test.go",
		"internal/tool/tool_test.go",
		"internal/cli/cli_test.go",
	}))
}
