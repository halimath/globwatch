# globwatch

A file system glob watcher for golang. 

![CI Status][ci-img-url] 
[![Go Report Card][go-report-card-img-url]][go-report-card-url] 
[![Package Doc][package-doc-img-url]][package-doc-url] 
[![Releases][release-img-url]][release-url]

[ci-img-url]: https://github.com/halimath/globwatch/workflows/CI/badge.svg
[go-report-card-img-url]: https://goreportcard.com/badge/github.com/halimath/globwatch
[go-report-card-url]: https://goreportcard.com/report/github.com/halimath/globwatch
[package-doc-img-url]: https://img.shields.io/badge/GoDoc-Reference-blue.svg
[package-doc-url]: https://pkg.go.dev/github.com/halimath/globwatch
[release-img-url]: https://img.shields.io/github/v/release/halimath/globwatch.svg
[release-url]: https://github.com/halimath/globwatch/releases

`globwatch` provides a file system watcher that detects changes recursively for
files that match a glob pattern.

# Installation

`globwatch` is provided as a go module and requires go >= 1.18.

```shell
go get github.com/halimath/globwatch@main
```

# Usage

## Creating a Watcher

`globwatch` provides a `Watcher` that can be created using the `New` function.
The function receives a `fs.FS` handle to watch, a pattern (see below) and
a check interval.

Once a `Watcher` has been created, you can start the watch routine by invoking
`Start` or `StartContext`. The second function expects a `context.Context`
that - when done - causes the watcher to terminate. Otherwise you can invoke
`Close` to finish watching. Once finished a `Watcher` cannot be restarted.

```go
watcher, err := globwatch.New(fsys, "**/*_test.go", time.Millisecond)
if err != nil {
    // ...
}

if err := watcher.Start(); err != nil {
    // ...
}

// ..

watcher.Close()
```

## Receiving changes

A `Watcher` communicates changes via a channel. The channel is available via
the `C` method.

```go
for e := range watcher.C() {
    fmt.Printf("%8s %s\n", e.Type, e.Path)
}
```

In addition you can subscribe for errors by reading from an `error`s channel
available via the `ErrorsChan` method.

## Pattern format

The pattern format used by `globwatch` works similar to the 
[pattern format of `.gitignore`](https://git-scm.com/docs/gitignore). It is
completely compatible with the pattern format used by `os.Glob` or `fs.Glob`
and extends it.

The format is specified as the following EBNF:

```ebnf
pattern = term, { '/', term };

term        = '**' | name;
name        = { charSpecial | group | escapedChar | '*' | '?' };
charSpecial = (* any unicode rune except '/', '*', '?', '[' and '\' *);
char        = (* any unicode rune *);
escapedChar = '\\', char;
group       = '[', [ '^' ] { escapedChar | groupChar | range } ']';
groupChar   = (* any unicode rune except '-' and ']' *);
range       = ( groupChar | escapedChar ), '-', (groupChar | escapedChar);
```

The format operators have the following meaning:

* any character (rune) matches the exactly this rune - with the following
  exceptions
* `/` works as a directory separator. It matches directory boundarys of the
  underlying system independently of the separator char used by the OS.
* `?` matches exactly one non-separator char
* `*` matches any number of non-separator chars - including zero
* `\` escapes a character's special meaning allowing `*` and `?` to be used
  as regular characters.
* `**` matches any number of nested directories. If anything is matched it
  always extends until a separator or the end of the name.
* Groups can be defined using the `[` and `]` characters. Inside a group the
  special meaning of the characters mentioned before is disabled but the
  following rules apply
    * any character used as part of the group acts as a choice to pick from
    * if the group's first character is a `^` the whole group is negated
    * a range can be defined using `-` matching any rune between low and high
      inclusive
    * Multiple ranges can be given. Ranges can be combined with choices.
    * The meaning of `-` and `]` can be escacped using `\`

# Performance

`globwatch` separates pattern parsing and matching. This can create a 
performance benefit when applied repeatedly. When reusing a precompiled pattern
to match filenames `globwatch` outperforms `filepath.Match` with both simple
and complex patterns. When not reusing the parsed pattern, `filepath` works
much faster (but lacks the additional features).

Test | Execution time `[ns/op]` | Memory usage `[B/op]` | Allocations per op
-- | --: | --: | --:
`filepath` simple pattern                        |   15.5 | 0    | 0
`globwatch` simple pattern (reuse)               |    3.9 | 0    | 0
`globwatch` simple pattern (noreuse)             |  495.0 | 1112 | 5
`filepath` complex pattern                       |  226.2 |    0 | 0
`globwatch` complex pattern (reuse)              |  108.1 |    0 | 0
`globwatch` complex pattern (noreuse)            | 1103.0 | 2280 | 8
`globwatch` directory wildcard pattern (reuse)   |  111.7 |    0 | 0
`globwatch` directory wildcard pattern (noreuse) | 1229.0 | 2280 | 8

# License

Copyright 2022 Alexander Metzner.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
