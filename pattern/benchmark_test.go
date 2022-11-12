package pattern

import (
	"path/filepath"
	"testing"
)

const (
	filename                 = "bar/foo_test.go"
	simplePattern            = "foo*.go"
	complexPattern           = "*/[a-f]oo[^a-z]*.go"
	directoryWildcardPattern = "**/[a-f]oo[^a-z]*.go"
)

func BenchmarkGlobwatch_simple_reuse(b *testing.B) {
	p, err := New(simplePattern)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Match(filename)
	}
}

func BenchmarkGlobwatch_simple_noreuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p, err := New(simplePattern)
		if err != nil {
			b.Fatal(err)
		}
		p.Match("bar/foo_test.go")
	}
}

func BenchmarkFilepath_simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filepath.Match(simplePattern, "bar/foo_test.go")
	}
}

func BenchmarkGlobwatch_complex_reuse(b *testing.B) {
	p, err := New(complexPattern)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Match(filename)
	}
}

func BenchmarkGlobwatch_complex_noreuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p, err := New(complexPattern)
		if err != nil {
			b.Fatal(err)
		}
		p.Match("bar/foo_test.go")
	}
}

func BenchmarkFilepath_complex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filepath.Match(complexPattern, "bar/foo_test.go")
	}
}

func BenchmarkGlobwatch_directoryWildcard_reuse(b *testing.B) {
	p, err := New(directoryWildcardPattern)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Match(filename)
	}
}

func BenchmarkGlobwatch_directoryWildcard_noreuse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p, err := New(directoryWildcardPattern)
		if err != nil {
			b.Fatal(err)
		}
		p.Match("bar/foo_test.go")
	}
}
