package globwatch

import (
	"testing"
	"time"

	"github.com/halimath/fsmock"

	. "github.com/halimath/expect-go"
)

func TestWatcher_detecChanges(t *testing.T) {
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

	watcher, err := New(fsys, "**/*_test.go", time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if err := watcher.determineInitialState(); err != nil {
		t.Fatal(err)
	}

	fsys.Touch("go.mod")
	fsys.Touch("cmd/main_test.go")

	watcher.detectChanges()

	fsys.Touch("cmd/main_test.go")
	fsys.Touch("internal/tool_test.go")

	watcher.detectChanges()

	fsys.Rm("internal")

	watcher.detectChanges()

	close(watcher.c)

	evts := make([]Event, 0, 20)
	for evt := range watcher.c {
		evts = append(evts, evt)
	}

	ExpectThat(t, evts).Is(DeepEqual([]Event{
		{
			Type: Created,
			Path: "cmd/main_test.go",
		},
		{
			Type: Modified,
			Path: "cmd/main_test.go",
		},
		{
			Type: Modified,
			Path: "internal/tool_test.go",
		},
		{
			Type: Deleted,
			Path: "internal/tool_test.go",
		},
	}))
}

func TestEventType_String(t *testing.T) {
	tests := map[EventType]string{
		Created:       "created",
		Deleted:       "deleted",
		Modified:      "modified",
		EventType(99): "unknown",
	}

	for in, want := range tests {
		ExpectThat(t, in.String()).Is(Equal(want))
	}
}
