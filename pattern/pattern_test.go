package pattern

import (
	"errors"
	"testing"

	. "github.com/halimath/expect-go"
	"github.com/halimath/fsmock"
)

func TestNew(t *testing.T) {
	tab := map[string]error{
		"main.go":         nil,
		"main_test.go":    nil,
		"foo/foo_test.go": nil,
		"?.go":            nil,
		"*.go":            nil,
		"**/*.go":         nil,

		"//":     ErrInvalidPattern,
		"foo//":  ErrInvalidPattern,
		"*?.go":  ErrInvalidPattern,
		"?*.go":  ErrInvalidPattern,
		"**?.go": ErrInvalidPattern,
		"**f":    ErrInvalidPattern,
	}

	for in, want := range tab {
		_, got := New(in)
		if want != nil && !errors.Is(got, want) {
			t.Errorf("%q: wanted %v but got %v", in, want, got)
		} else if want == nil && got != nil {
			t.Errorf("%q: wanted nil but got %v", in, got)
		}
	}
}

func TestPattern_Match_literal(t *testing.T) {
	p, err := New("foo/bar.go")
	if err != nil {
		t.Fatal(err)
	}

	tab := map[string]bool{
		"foo.go":       false,
		"foo/a.go":     false,
		"foo/bar.go/x": false,
		"bar.go":       false,
		"foo/bar.go":   true,
	}

	for in, want := range tab {
		got := p.Match(in)
		if want != got {
			t.Errorf("%q: wanted %v but got %v", in, want, got)
		}
	}
}

func TestPattern_Match_single(t *testing.T) {
	p, err := New("foo/m?.go")
	if err != nil {
		t.Fatal(err)
	}

	tab := map[string]bool{
		"mi.go":       false,
		"foo.go":      false,
		"foo/a.go":    false,
		"foo/ma.go/x": false,
		"foo/m/":      false,
		"foo/ma.go":   true,
		"foo/mu.go":   true,
		"foo/mx.go":   true,
		"foo/mä.go":   true,
	}

	for in, want := range tab {
		got := p.Match(in)
		if want != got {
			t.Errorf("%q: wanted %v but got %v", in, want, got)
		}
	}
}

func TestPattern_Match_any(t *testing.T) {
	p, err := New("foo/m*.go")
	if err != nil {
		t.Fatal(err)
	}

	tab := map[string]bool{
		"mi.go":       false,
		"foo.go":      false,
		"foo/a.go":    false,
		"foo/ma.go/x": false,
		"foo/ma.go":   true,
		"foo/mu.go":   true,
		"foo/mx.go":   true,
		"foo/m.go":    true,
		"foo/mäx.go":  true,
	}

	for in, want := range tab {
		got := p.Match(in)
		if want != got {
			t.Errorf("%q: wanted %v but got %v", in, want, got)
		}
	}
}

func TestPattern_Match_anyDir(t *testing.T) {
	p, err := New("**/m.go")
	if err != nil {
		t.Fatal(err)
	}

	tab := map[string]bool{
		"foo.go":       false,
		"foo/a.go":     false,
		"m.go":         true,
		"foo/m.go":     true,
		"bar/m.go":     true,
		"foo/bar/m.go": true,
	}

	for in, want := range tab {
		got := p.Match(in)
		if want != got {
			t.Errorf("%q: wanted %v but got %v", in, want, got)
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
