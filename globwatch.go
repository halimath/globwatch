// Package globwatch implements a filesystem watcher that supports recursive
// watching using an extended glob pattern syntax.
//
// The glob pattern syntax is descibed in github.com/halimath/globwatch/pattern.
//
// The watcher is implemented based on a directory polling which periodically
// uses fs.WalkDir to walk a directory and check each file for changes.
// The watcher does not rely on kernel support like inotify or kqueue. The
// decision to work around these kernel features was made to support a large
// number of files and directories to watch. Especially with kqueue on MacOS
// you can quickly hit the open files limit.
package globwatch

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/halimath/globwatch/pattern"
)

// EventType defines the type of event for a changed file.
type EventType int

const (
	// Created reports that a new file has been created.
	Created EventType = iota + 1
	// Modified reports that an existing file has changed.
	Modified
	// Deleted reports that an existing file has been deleted (or moved away).
	Deleted
)

// String returns a string representation of t.
func (t EventType) String() string {
	switch t {
	case Created:
		return "created"
	case Modified:
		return "modified"
	case Deleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// Event models a single event reported by a Watcher for a single file.
type Event struct {
	// The event's type
	Type EventType
	// The full path of the file relative to the watched root
	Path string
}

// Watcher implements glob watching. Events for changed files will be reported
// via C. Any error that occured during change detection will be reported vi
// Errors. Make sure you consume both channels or you will block change
// detection otherwise.
type Watcher struct {
	fsys     fs.FS
	pat      *pattern.Pattern
	interval time.Duration
	modtimes map[string]time.Time
	close    chan struct{}
	closed   chan struct{}
	errors   chan error
	c        chan Event
}

// New creates a new watcher. The watcher will use fsys to access the files
// and directories. It will use fsys as the root to watch. pat defines the
// pattern relative to fsys' root. interval defines how often to check for
// changes.
// A created watcher will not start watching for changes unless Start or
// StartContext is called.
func New(fsys fs.FS, pat string, interval time.Duration) (*Watcher, error) {
	p, err := pattern.New(pat)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		modtimes: make(map[string]time.Time),
		fsys:     fsys,
		pat:      p,
		interval: interval,
		close:    make(chan struct{}),
		closed:   make(chan struct{}),
		errors:   make(chan error, 10),
		c:        make(chan Event, 10),
	}, nil
}

// C returns a channel used to receive change Events.
func (w *Watcher) C() <-chan Event {
	return w.c
}

// ErrorsChan returns a channel used to receive errors during watching.
func (w *Watcher) ErrorsChan() <-chan error {
	return w.errors
}

// Start starts watching using a default context. See StartContext.
func (w *Watcher) Start() error {
	return w.StartContext(context.Background())
}

// StartContext starts watching for changes. If ctx will be canceled w will
// be closed. The funtion reports any error that occured during initial
// file analysis.
func (w *Watcher) StartContext(ctx context.Context) error {
	if err := w.determineInitialState(); err != nil {
		return err
	}

	ticker := time.NewTicker(w.interval)

	go func() {
		defer ticker.Stop()
		defer close(w.c)
		defer close(w.errors)
		defer close(w.closed)

		for {
			select {
			case <-ticker.C:
				w.detectChanges()
			case <-w.close:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Close closes w. The change detection goroutine will be shutdown gracefully
// and both w.C and w.Errors will be closed before Close returns.
func (w *Watcher) Close() {
	close(w.close)
	<-w.closed
}

func (w *Watcher) determineInitialState() error {
	names, err := w.pat.GlobFS(w.fsys, ".")
	if err != nil {
		return fmt.Errorf("failed to detect watcher: %w", err)
	}

	for _, name := range names {
		i, err := fs.Stat(w.fsys, name)
		if err != nil {
			w.errors <- err
			continue
		}
		w.modtimes[name] = i.ModTime()
	}

	return nil
}

func (w *Watcher) detectChanges() {
	names, err := w.pat.GlobFS(w.fsys, ".")
	if err != nil {
		w.errors <- fmt.Errorf("failed to detect changes: %w", err)
		return
	}

	foundNames := make(map[string]struct{})

	for _, name := range names {
		foundNames[name] = struct{}{}

		i, err := fs.Stat(w.fsys, name)
		if err != nil {
			w.errors <- err
			continue
		}

		got, ok := w.modtimes[name]
		if !ok {
			w.modtimes[name] = i.ModTime()
			w.c <- Event{
				Type: Created,
				Path: name,
			}

			continue
		}

		if i.ModTime().After(got) {
			w.modtimes[name] = i.ModTime()
			w.c <- Event{
				Type: Modified,
				Path: name,
			}
		}
	}

	for n := range w.modtimes {
		if _, ok := foundNames[n]; !ok {
			delete(w.modtimes, n)
			w.c <- Event{
				Type: Deleted,
				Path: n,
			}
		}
	}
}
