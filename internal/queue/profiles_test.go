package queue

import (
	"dl-manager/internal/model"
	"testing"
	"time"
)

func TestProfileMutationPreventsConcurrentDispatch(t *testing.T) {
	r := &testRunner{started: make(chan model.Job, 4)}
	m, s := setup(t, r)
	s.PutProfile(model.Profile{ID: "profile", Site: "generic"})
	guard, ok := any(m).(interface {
		WithProfileMutation(string, func() error) error
	})
	if !ok {
		t.Fatal("missing atomic profile mutation boundary")
	}
	j := model.Job{ID: model.ID(), URL: "https://example.org/photo.jpg", Site: "generic", ProfileID: "profile", Destination: t.TempDir(), Filters: model.DefaultFilters()}
	if err := m.Add(j); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	mutated := make(chan error, 1)
	go func() {
		mutated <- guard.WithProfileMutation("profile", func() error { close(entered); <-release; return nil })
	}()
	<-entered
	started := make(chan error, 1)
	go func() { started <- m.Start() }()
	select {
	case <-r.started:
		close(release)
		t.Fatal("dispatch overlapped credential replacement")
	case <-time.After(75 * time.Millisecond):
	}
	close(release)
	if err := <-mutated; err != nil {
		t.Fatal(err)
	}
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return r.active.Load() == 1 })
	called := false
	if err := guard.WithProfileMutation("profile", func() error { called = true; return nil }); err == nil || called {
		t.Fatal("active profile was modified")
	}
}
