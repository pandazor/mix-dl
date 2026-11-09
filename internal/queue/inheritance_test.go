package queue

import (
	"dl-manager/internal/model"
	"testing"
	"time"
)

func TestDefaultAccountResolvedAfterJobCreation(t *testing.T) {
	for _, mode := range []string{"inherit", "", "none", "explicit"} {
		t.Run("mode-"+mode, func(t *testing.T) {
			r := &testRunner{started: make(chan model.Job, 2)}
			m, s := setup(t, r)
			j := model.Job{ID: model.ID(), URL: "https://rule34.xxx/index.php?page=post&s=list&tags=landscape", Site: "rule34", ProfileMode: mode, Destination: t.TempDir(), Filters: model.DefaultFilters()}
			if mode == "explicit" {
				j.ProfileID = "chosen"
				s.PutProfile(model.Profile{ID: "chosen", Site: "rule34", Method: "api"})
			}
			if e := m.Add(j); e != nil {
				t.Fatal(e)
			}
			if e := s.PutProfileWithDefault(model.Profile{ID: "new-account", Site: "rule34", Method: "api"}, true); e != nil {
				t.Fatal(e)
			}
			if e := m.Start(); e != nil {
				t.Fatal(e)
			}
			select {
			case got := <-r.started:
				want := "new-account"
				if mode == "none" {
					want = ""
				}
				if mode == "explicit" {
					want = "chosen"
				}
				if got.ProfileID != want {
					t.Fatalf("started with profile %q, want %q", got.ProfileID, want)
				}
				if want != "" && !m.ProfileBusy(want) {
					t.Fatal("resolved profile not locked")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("job not dispatched")
			}
		})
	}
}
func TestInheritedProfileChangesOnManualResume(t *testing.T) {
	r := &testRunner{started: make(chan model.Job, 4)}
	m, s := setup(t, r)
	s.PutProfileWithDefault(model.Profile{ID: "one", Site: "rule34"}, true)
	j := model.Job{ID: model.ID(), URL: "https://rule34.xxx/", Site: "rule34", ProfileMode: "inherit", Destination: t.TempDir(), Filters: model.DefaultFilters()}
	m.Add(j)
	m.Start()
	<-r.started
	m.Pause()
	await(t, func() bool { state, _ := m.Snapshot(); return state.Active == 0 })
	s.PutProfileWithDefault(model.Profile{ID: "two", Site: "rule34"}, true)
	m.Start()
	select {
	case got := <-r.started:
		if got.ProfileID != "two" {
			t.Fatal("resume used old default")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no resume")
	}
}
