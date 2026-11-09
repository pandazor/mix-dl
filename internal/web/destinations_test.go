package web

import (
	"context"
	"dl-manager/internal/model"
	"dl-manager/internal/queue"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

type destinationRunner struct{ started chan model.Job }

func (r *destinationRunner) Run(ctx context.Context, j model.Job, emit func(model.Event)) error {
	r.started <- j
	<-ctx.Done()
	return ctx.Err()
}
func TestDestinationInheritanceAtFirstStartAndPinnedOnResume(t *testing.T) {
	for _, mode := range []string{"global", "site", "explicit"} {
		t.Run(mode, func(t *testing.T) {
			a, _ := profileAPI(t)
			a.Manager.Close()
			runner := &destinationRunner{make(chan model.Job, 2)}
			a.Manager = queue.New(a.Store, runner)
			defer a.Manager.Close()
			initial := filepath.Join(t.TempDir(), "initial")
			global := filepath.Join(t.TempDir(), "global")
			site := filepath.Join(t.TempDir(), "site")
			explicit := filepath.Join(t.TempDir(), "explicit")
			state, _ := a.Store.Snapshot()
			cfg := state.Settings
			cfg.Destination = initial
			if e := a.Manager.SaveSettings(cfg); e != nil {
				t.Fatal(e)
			}
			req := map[string]any{"urls": []string{"https://example.com/fixture.jpg"}}
			if mode == "explicit" {
				req["destination"] = explicit
			}
			b, _ := json.Marshal(req)
			w := request(t, a, "POST", "jobs", string(b))
			if w.Code != 200 {
				t.Fatal(w.Body.String())
			}
			cfg.Destination = global
			if mode == "site" {
				cfg.Sites["generic"] = model.SiteSettings{Destination: site}
			}
			if e := a.Manager.SaveSettings(cfg); e != nil {
				t.Fatal(e)
			}
			want := global
			if mode == "site" {
				want = site
			}
			if mode == "explicit" {
				want = explicit
			}
			preview, _ := a.Manager.Snapshot()
			if preview.Jobs[0].Destination != want {
				t.Errorf("preview destination=%q want %q", preview.Jobs[0].Destination, want)
			}
			if e := a.Manager.Start(); e != nil {
				t.Fatal(e)
			}
			select {
			case got := <-runner.started:
				if got.Destination != want {
					t.Fatalf("started at %q want %q", got.Destination, want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("not started")
			}
			a.Manager.Pause()
			deadline := time.Now().Add(3 * time.Second)
			for {
				st, _ := a.Manager.Snapshot()
				if st.Active == 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("pause timeout")
				}
				time.Sleep(time.Millisecond * 10)
			}
			cfg.Destination = filepath.Join(t.TempDir(), "changed-again")
			cfg.Sites = map[string]model.SiteSettings{}
			a.Manager.SaveSettings(cfg)
			if e := a.Manager.Start(); e != nil {
				t.Fatal(e)
			}
			select {
			case got := <-runner.started:
				if got.Destination != want {
					t.Fatalf("resume changed folder: %q", got.Destination)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("no resume")
			}
		})
	}
}
func TestLegacyDestinationRefreshIsExplicitAtomicAndPaused(t *testing.T) {
	a, _ := profileAPI(t)
	state, _ := a.Store.Snapshot()
	old := filepath.Join(t.TempDir(), "old")
	global := state.Settings.Destination
	for _, id := range []string{"untouched", "started"} {
		j := model.Job{ID: id, URL: "https://example.com/" + id + ".jpg", Site: "generic", Destination: old, Filters: model.DefaultFilters(), Status: "queued"}
		if id == "started" {
			j.Started = model.Now()
		}
		if e := a.Store.PutJob(j); e != nil {
			t.Fatal(e)
		}
	}
	w := request(t, a, "POST", "jobs/destinations", `{"ids":["untouched","started"]}`)
	if w.Code == 200 {
		t.Fatal("accepted started task")
	}
	st, _ := a.Store.Snapshot()
	for _, j := range st.Jobs {
		if j.Destination != old {
			t.Fatal("partial mutation")
		}
	}
	w = request(t, a, "POST", "jobs/destinations", `{"ids":["untouched"]}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	st, _ = a.Store.Snapshot()
	for _, j := range st.Jobs {
		want := old
		if j.ID == "untouched" {
			want = global
		}
		if j.Destination != want {
			t.Fatalf("job %s destination %q want %q", j.ID, j.Destination, want)
		}
	}
	if st.Running {
		t.Fatal("refresh started queue")
	}
}
