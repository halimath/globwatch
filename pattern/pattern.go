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
	// Backslash escapes the next character's special meaning
	Backslash = '\\'
	// GroupStart starts a range
	GroupStart = '['
	// GroupEnd starts a range
	GroupEnd = ']'
	// GroupNegate when used as the first character of a group negates the group.
	GroupNegate = '^'
	// Range defines the range operator
	Range = '-'
)

var (
	// ErrBadPattern is returned when an invalid pattern is found. Make
	// sure you use errors.Is to compare errors to this sentinel value.
	ErrBadPattern = errors.New("bad pattern")
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
			if len(tokens) > 0 && tokens[len(tokens)-1].r == Separator {
				return nil, fmt.Errorf("%w: unexpected //", ErrBadPattern)
			}
			t = token{tokenTypeLiteral, Separator, runeGroup{}}

		case SingleWildcard:
			if len(tokens) > 0 && (tokens[len(tokens)-1].t == tokenTypeAnyRunes || tokens[len(tokens)-1].t == tokenTypeAnyDirectories) {
				return nil, fmt.Errorf("%w: unexpected ?", ErrBadPattern)
			}
			t = token{tokenTypeSingleRune, 0, runeGroup{}}

		case AnyWildcard:
			if len(tokens) > 0 && (tokens[len(tokens)-1].t == tokenTypeSingleRune || tokens[len(tokens)-1].t == tokenTypeAnyDirectories) {
				return nil, fmt.Errorf("%w: unexpected ?", ErrBadPattern)
			}

			t = token{tokenTypeAnyRunes, 0, runeGroup{}}

			if len(p[l:]) > 0 {
				n, nl := utf8.DecodeRuneInString(p[l:])
				if n == AnyWildcard {
					d, _ := utf8.DecodeRuneInString(p[l+nl:])
					if d != Separator {
						return nil, fmt.Errorf("%w: unexpected %c after **", ErrBadPattern, d)
					}

					t.t = tokenTypeAnyDirectories
					l += nl
				}
			}

		case Backslash:
			if len(p[l:]) == 0 {
				return nil, fmt.Errorf("%w: no character given after \\", ErrBadPattern)
			}

			p = p[l:]
			r, l = utf8.DecodeRuneInString(p)

			t = token{tokenTypeLiteral, r, runeGroup{}}

		case GroupStart:
			var err error
			t, l, err = parseGroup(p)
			if err != nil {
				return nil, err
			}

		case GroupEnd:
			return nil, fmt.Errorf("%w: using ] w/o [", ErrBadPattern)

		default:
			t = token{tokenTypeLiteral, r, runeGroup{}}
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

func parseGroup(p string) (token, int, error) {
	// re-read the [. No need to assert the rune here as it has been
	// done in the main parsing loop.
	_, le := utf8.DecodeRuneInString(p)
	t := token{
		t: tokenTypeGroup,
		g: runeGroup{},
	}

	initialLen := le
	var start rune

	for {
		if len(p[le:]) == 0 {
			return t, le, fmt.Errorf("%w: missing %c", ErrBadPattern, GroupEnd)
		}

		r, l := utf8.DecodeRuneInString(p[le:])
		le += l

		if initialLen == le-l && r == GroupNegate {
			t.g.neg = true
			continue
		}

		switch r {
		case GroupEnd:
			if start != 0 {
				t.g.runes = append(t.g.runes, start)
			}

			return t, le, nil

		case Range:
			if start == 0 {
				return t, le, fmt.Errorf("%w: missing start for character range", ErrBadPattern)
			}

			if len(p[le:]) == 0 {
				return t, le, fmt.Errorf("%w: missing range end", ErrBadPattern)
			}

			r, l = utf8.DecodeRuneInString(p[le:])
			le += l

			switch r {
			case GroupEnd:
				return t, le, fmt.Errorf("%w: unterminated range", ErrBadPattern)

			case Backslash:
				if len(p[le:]) == 0 {
					return t, le, fmt.Errorf("%w: missing character after \\", ErrBadPattern)
				}
				r, l = utf8.DecodeRuneInString(p[le:])
				le += l
				fallthrough

			default:
				t.g.ranges = append(t.g.ranges, runeRange{start, r})
				start = 0
			}

		case Backslash:
			if len(p[le:]) == 0 {
				return t, le, fmt.Errorf("%w: missing character after \\", ErrBadPattern)
			}

			r, l = utf8.DecodeRuneInString(p[le:])
			le += l
			fallthrough

		default:
			if start != 0 {
				t.g.runes = append(t.g.runes, start)
			}
			start = r
		}
	}
}

// match is used internally to implement a simple recursive backtracking
// algorithmn using the token list t to match against file path f.
func match(f string, t []token) bool {
	for {
		if len(f) == 0 {
			if len(t) == 0 {
				return true
			}

			if len(t) == 1 && t[0].t == tokenTypeAnyRunes {
				return true
			}

			return false
		}

		if len(t) == 0 {
			return false
		}

		r, le := utf8.DecodeRuneInString(f)

		switch t[0].t {
		case tokenTypeLiteral:
			if t[0].r != r {
				return false
			}

		case tokenTypeGroup:
			if !t[0].g.match(r) {
				return false
			}

		case tokenTypeSingleRune:
			if r == Separator {
				return false
			}

		case tokenTypeAnyRunes:
			if r == Separator {
				return match(f, t[1:])
			}

			if match(f[le:], t) {
				return true
			}

			if match(f, t[1:]) {
				return true
			}

		case tokenTypeAnyDirectories:
			if match(f, t[2:]) {
				return true
			}

			var l2 int
			for {
				if len(f[le+l2:]) == 0 {
					return false
				}

				n, nl := utf8.DecodeRuneInString(f[le+l2:])
				l2 += nl

				if n == Separator {
					break
				}
			}

			if match(f[le+l2:], t[2:]) {
				return true
			}

			return match(f[le+l2:], t)
		}

		t = t[1:]
		f = f[le:]
	}
}

// tokenType enumerates the different types of tokens.
type tokenType int

const (
	// a rune literal
	tokenTypeLiteral tokenType = iota + 1
	// any single non-separator rune
	tokenTypeSingleRune
	// any number of non-separator runes (incl. zero)
	tokenTypeAnyRunes
	// any number runes including separators. Matches whole directories.
	tokenTypeAnyDirectories
	// a group of rune consisting of named runes and/or ranges. Might be negated.
	tokenTypeGroup
)

// token implements a single token in the pattern.
type token struct {
	// the token's type
	t tokenType
	// a literal rune to matche. Literal runes are stored separate from groups
	// to improve matching performance.
	r rune
	// A rune group to match.
	g runeGroup
}

// A group of runes. Groups can contain any number of enumerated runes and rune
// ranges. In addition a whole group can be negated.
type runeGroup struct {
	// Whether the group is negated
	neg bool
	// Enumerated runes contained in this group
	runes []rune
	// All ranges contained in this group
	ranges []runeRange
}

// match matches r with g. It returns true if r is matched.
func (g runeGroup) match(r rune) bool {
	for _, ru := range g.runes {
		if ru == r {
			return !g.neg
		}
	}

	for _, rang := range g.ranges {
		if rang.match(r) {
			return !g.neg
		}
	}

	return g.neg
}

// A closed range of runes consisting of all runes between lo and hi both
// inclusive.
type runeRange struct {
	lo, hi rune
}

// match returns whether r is in rg.
func (rg runeRange) match(r rune) bool {
	return rg.lo <= r && r <= rg.hi
}
