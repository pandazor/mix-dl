package web

import (
	"context"
	"dl-manager/internal/auth"
	"dl-manager/internal/model"
	"dl-manager/internal/queue"
	"dl-manager/internal/storage"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type idleRunner struct{}

func (idleRunner) Run(ctx context.Context, j model.Job, e func(model.Event)) error { return nil }
func TestAPIRequiresTokenAndPersistsAddedJob(t *testing.T) {
	dir := t.TempDir()
	s, e := storage.Open(filepath.Join(dir, "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	v, e := auth.New(filepath.Join(dir, "credentials"))
	if e != nil {
		t.Fatal(e)
	}
	m := queue.New(s, idleRunner{})
	defer m.Close()
	a := New(s, m, v, dir, dir)
	cfg := model.DefaultSettings()
	cfg.Destination = t.TempDir()
	m.SaveSettings(cfg)
	r := httptest.NewRequest("POST", "/api/queue/start", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatal("missing CSRF rejected", w.Code)
	}
	r = httptest.NewRequest("POST", "/api/jobs", strings.NewReader(`{"urls":["https://example.com/a.jpg"]}`))
	r.Header.Set("X-DL-Token", a.Token)
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	state, _ := m.Snapshot()
	if len(state.Jobs) != 1 || state.Running {
		t.Fatal("add lost job or auto started")
	}
}
