package queue

import (
	"context"
	"dl-manager/internal/model"
	"testing"
)

type slowStopRunner struct {
	started   chan struct{}
	cancelled chan struct{}
	release   chan struct{}
}

func (r *slowStopRunner) Run(ctx context.Context, j model.Job, emit func(model.Event)) error {
	r.started <- struct{}{}
	<-ctx.Done()
	r.cancelled <- struct{}{}
	<-r.release
	return ctx.Err()
}
func TestStartRejectsPendingCancellation(t *testing.T) {
	r := &slowStopRunner{make(chan struct{}, 2), make(chan struct{}, 2), make(chan struct{})}
	m, _ := setup(t, r)
	defer close(r.release)
	add(t, m, "generic")
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	<-r.started
	m.Pause()
	<-r.cancelled
	if err := m.Start(); err == nil {
		t.Fatal("resume accepted while stopping: it will be silently lost")
	}
	s, _ := m.Snapshot()
	if s.Running {
		t.Fatal("queue resumed during pending cancellation")
	}
}
