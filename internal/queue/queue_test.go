package queue

import (
	"context"
	"dl-manager/internal/model"
	"dl-manager/internal/storage"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type testRunner struct {
	active    atomic.Int32
	max       atomic.Int32
	started   chan model.Job
	emitBytes int64
}

func (r *testRunner) Run(ctx context.Context, j model.Job, emit func(model.Event)) error {
	n := r.active.Add(1)
	defer r.active.Add(-1)
	for {
		old := r.max.Load()
		if old >= n || r.max.CompareAndSwap(old, n) {
			break
		}
	}
	r.started <- j
	if r.emitBytes > 0 {
		emit(model.Event{Kind: "bytes", Bytes: r.emitBytes})
	}
	<-ctx.Done()
	return ctx.Err()
}
func await(t *testing.T, f func() bool) {
	t.Helper()
	limit := time.After(4 * time.Second)
	for !f() {
		select {
		case <-limit:
			t.Fatal("condition timed out")
		case <-time.After(10 * time.Millisecond):
		}
	}
}
func setup(t *testing.T, r Runner) (*Manager, *storage.Store) {
	t.Helper()
	s, e := storage.Open(filepath.Join(t.TempDir(), "s.db"))
	if e != nil {
		t.Fatal(e)
	}
	m := New(s, r)
	t.Cleanup(func() { m.Close(); s.Close() })
	return m, s
}
func add(t *testing.T, m *Manager, site string) {
	t.Helper()
	if e := m.Add(model.Job{ID: model.ID(), URL: "https://example.org/" + model.ID(), Site: site, Destination: t.TempDir(), Filters: model.DefaultFilters(), Status: "queued"}); e != nil {
		t.Fatal(e)
	}
}
func TestNoAutostartAndConcurrencyCap(t *testing.T) {
	r := &testRunner{started: make(chan model.Job, 10)}
	m, s := setup(t, r)
	v := model.DefaultSettings()
	v.Concurrency = 2
	s.SaveSettings(v)
	for range 5 {
		add(t, m, "generic")
	}
	time.Sleep(150 * time.Millisecond)
	if r.active.Load() != 0 {
		t.Fatal("auto started")
	}
	if e := m.Start(); e != nil {
		t.Fatal(e)
	}
	await(t, func() bool { return r.active.Load() == 2 })
	m.Pause()
	await(t, func() bool { return r.active.Load() == 0 })
	if r.max.Load() > 2 {
		t.Fatal("exceeded worker limit")
	}
}
func TestQuotaPauseResetDoesNotRestart(t *testing.T) {
	r := &testRunner{started: make(chan model.Job, 10), emitBytes: 110}
	m, s := setup(t, r)
	v := model.DefaultSettings()
	v.QuotaBytes = 100
	s.SaveSettings(v)
	add(t, m, "generic")
	m.Start()
	await(t, func() bool { v, _ := m.Snapshot(); return v.UsedBytes == 110 && !v.Running && v.Active == 0 })
	if e := m.Start(); e == nil {
		t.Fatal("quota allowed restart")
	}
	if e := m.ResetBudget(); e != nil {
		t.Fatal(e)
	}
	state, _ := m.Snapshot()
	if state.UsedBytes != 0 || state.Running {
		t.Fatal("reset resumed or did not reset")
	}
}
func TestSharedDestinationSerializesAndResetKeepsFiles(t *testing.T) {
	r := &testRunner{started: make(chan model.Job, 10)}
	m, s := setup(t, r)
	v := model.DefaultSettings()
	v.Concurrency = 3
	s.SaveSettings(v)
	dest := t.TempDir()
	for range 3 {
		m.Add(model.Job{ID: model.ID(), URL: "https://example.org/" + model.ID(), Site: "generic", Destination: dest, Filters: model.DefaultFilters(), Status: "queued"})
	}
	m.Start()
	await(t, func() bool { return r.active.Load() == 1 })
	time.Sleep(150 * time.Millisecond)
	if r.max.Load() != 1 {
		t.Fatal("shared archive overlap")
	}
	m.ResetBudget()
	vstate, _ := m.Snapshot()
	if len(vstate.Jobs) != 3 || vstate.Running {
		t.Fatal("reset lost entries or resumed")
	}
}
