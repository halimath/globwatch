package pattern

import (
	"path/filepath"
	"testing"
)

func BenchmarkPattern(b *testing.B) {
	p, err := New("**/foo*.go")
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		p.Match("bar/foo_test.go")
	}
}

func BenchmarkFilepathMatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filepath.Match("*/foo*.go", "bar/foo_test.go")
	}
}
