package globwatch_test

import (
	"testing"
	"time"

	"github.com/halimath/fsmock"
	"github.com/halimath/globwatch"

	. "github.com/halimath/expect-go"
)

func TestWatcher(t *testing.T) {
	fsys := fsmock.New(fsmock.NewDir("",
		fsmock.EmptyFile("go.mod"),
		fsmock.EmptyFile("go.sum"),
		fsmock.NewDir("cmd",
			fsmock.TextFile("main.go", "package main"),
		),
		fsmock.NewDir("internal",
			fsmock.EmptyFile("tool.go"),
			fsmock.EmptyFile("tool_test.go"),
		),
	))

	watcher, err := globwatch.New(fsys, "**/*_test.go", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	if err := watcher.Start(); err != nil {
		t.Fatal(err)
	}

	evts := make([]globwatch.Event, 0, 20)

	go func() {
		for evt := range watcher.C() {
			evts = append(evts, evt)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	fsys.Touch("go.mod")
	fsys.Touch("cmd/main_test.go")

	time.Sleep(10 * time.Millisecond)

	ExpectThat(t, evts).Is(DeepEqual([]globwatch.Event{
		{
			Type: globwatch.Created,
			Path: "cmd/main_test.go",
		},
	}))

	watcher.Close()
}
