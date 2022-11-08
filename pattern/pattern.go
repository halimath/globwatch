// Package pattern implements a language for specifying glob patterns for path
// names starting at some root. The language does not follow the specs from
// filepath.Match but differs in one major point: it allows for directory
// wildcards.
//
// Patterns consist of normal characters, non-separator wildcards '*' and '?',
// separators '/' and directory wildcards '**'.
//
// A somewhat formal grammer can be given as:
//
//	pattern -> term ('/' term)*
//	term    -> '**' // directory wildcard: matches any directory
//	term    -> name
//	name    -> (char | '*' | '?')+
//	char    -> <any character except '/', '*' or '?'>
package pattern

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	// Separator defines the path separator to use in patterns. This is always
	// a forward slash independently of the underlying's OS separator
	Separator = '/'
	// SingleWildcard defines the the single non-separator character wildcard
	// operator.
	SingleWildcard = '?'
	// AnyWildcard defines the the any number of non-separator characters
	// wildcard operator.
	AnyWildcard = '*'
)

var (
	// ErrInvalidPattern is returned when an invalid pattern is found. Make
	// sure you use errors.Is to compare errors to this sentinel value.
	ErrInvalidPattern = errors.New("invalid pattern")
)

// Pattern defines a glob pattern prepared ahead of time which can be used to
// match filenames. Pattern is safe to use concurrently.
type Pattern struct {
	tokens []token
}

// New creates a new pattern from pat and returns it. It returns an error
// indicating any invalid pattern.
func New(pat string) (*Pattern, error) {
	var tokens []token

	p := pat
	for {
		if len(p) == 0 {
			return &Pattern{tokens: tokens}, nil
		}

		r, l := utf8.DecodeRuneInString(p)

		var t token
		switch r {
		case Separator:
			if len(tokens) > 0 && tokens[len(tokens)-1].t == separator {
				return nil, fmt.Errorf("%w: unexpected //", ErrInvalidPattern)
			}
			t = token{separator, r}

		case SingleWildcard:
			if len(tokens) > 0 && (tokens[len(tokens)-1].t == any || tokens[len(tokens)-1].t == anyDirectory) {
				return nil, fmt.Errorf("%w: unexpected ?", ErrInvalidPattern)
			}
			t = token{single, r}

		case AnyWildcard:
			if len(tokens) > 0 && (tokens[len(tokens)-1].t == single || tokens[len(tokens)-1].t == anyDirectory) {
				return nil, fmt.Errorf("%w: unexpected ?", ErrInvalidPattern)
			}

			t = token{any, r}

			if len(p[l:]) > 0 {
				n, nl := utf8.DecodeRuneInString(p[l:])
				if n == AnyWildcard {
					d, _ := utf8.DecodeRuneInString(p[l+nl:])
					if d != Separator {
						return nil, fmt.Errorf("%w: unexpected %c after **", ErrInvalidPattern, d)
					}

					t.t = anyDirectory
					l += nl
				}
			}

		default:
			t = token{literal, r}
		}

		tokens = append(tokens, t)
		p = p[l:]
	}
}

// Match matches a file's path name f to the compiled pattern and returns
// whether the path matches the pattern or not.
func (pat *Pattern) Match(f string) bool {
	return match(f, pat.tokens)
}

// GlobFS applies pat to all files found in fsys under root and returns the
// matching path names as a string slice. It uses fs.WalkDir internally and all
// constraints given for that function apply to GlobFS.
func (pat *Pattern) GlobFS(fsys fs.FS, root string) ([]string, error) {
	results := make([]string, 0)
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// TODO: Optimize with descend into checks
			return nil
		}

		if root != "." && root != "" {
			p = strings.Replace(p, root, "", 1)
		}

		if pat.Match(p) {
			results = append(results, p)
		}

		return nil
	})

	return results, err
}

// match is used internally to implement a simple recursive backtracking
// algorithmn using the token list t to match against file path f.
func match(f string, t []token) bool {
	for {
		if len(f) == 0 {
			return len(t) == 0
		}

		if len(t) == 0 {
			return false
		}

		r, l := utf8.DecodeRuneInString(f)

		switch t[0].t {
		case separator:
			if r != filepath.Separator {
				return false
			}

		case literal:
			if r != t[0].r {
				return false
			}

		case single:
			if r == filepath.Separator {
				return false
			}

		case any:
			if r == filepath.Separator {
				return false
			}

			if match(f[l:], t) {
				return true
			}

			if match(f, t[1:]) {
				return true
			}

		case anyDirectory:
			if match(f, t[2:]) {
				return true
			}

			var l2 int
			for {
				if len(f[l+l2:]) == 0 {
					return false
				}

				n, nl := utf8.DecodeRuneInString(f[l+l2:])
				l2 += nl

				if n == Separator {
					break
				}
			}

			if match(f[l+l2:], t[2:]) {
				return true
			}

			return match(f[l+l2:], t)
		}

		t = t[1:]
		f = f[l:]
	}
}

// tokenType enumerates the different types of tokens.
type tokenType int

const (
	// a separator in the pattern
	separator tokenType = iota + 1
	// a literal rune
	literal
	// any single non-separator rune
	single
	// any number of non-separator runes (incl. zero)
	any
	// any number runes including separators. Matches whole directories.
	anyDirectory
)

// token implements a single token in the pattern.
type token struct {
	t tokenType
	r rune
}
