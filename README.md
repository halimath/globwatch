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
[pattern format of `.gitignore`](https://git-scm.com/docs/gitignore). 

The format is defined as

```ebnf
pattern = term, { '/', term };

term    = '**' | name;
name    = { char | '*' | '?' };
char    = (* <any character except '/', '*' or '?'> *)
```

The format uses the following operators

Operator | Description
-- | --
`*` | Any number of characters (incl. zero) excluding the directory separator `/`
`?` | A single character excluding the directory separator `/`
`**` | Any number of "directories" (incl. zero). The `**` operator consumes whole directories up to the next `/`

# License

Copyright 2022 Alexander Metzner.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
