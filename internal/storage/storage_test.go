package storage

import (
	"dl-manager/internal/model"
	"path/filepath"
	"testing"
)

func TestPersistenceAndCrashRecovery(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.db")
	s, e := Open(p)
	if e != nil {
		t.Fatal(e)
	}
	j := model.Job{ID: "test", URL: "https://example.com/1", Status: "running", Destination: t.TempDir(), Filters: model.DefaultFilters()}
	if e = s.PutJob(j); e != nil {
		t.Fatal(e)
	}
	if e = s.AddUsage(41); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(p)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	state, e := s.Snapshot()
	if e != nil {
		t.Fatal(e)
	}
	if len(state.Jobs) != 1 || state.Jobs[0].Status != "interrupted" || state.UsedBytes != 41 {
		t.Fatalf("bad recovery %+v", state)
	}
	if e = s.ResetBudget(); e != nil {
		t.Fatal(e)
	}
	state, _ = s.Snapshot()
	if state.UsedBytes != 0 || len(state.Jobs) != 1 {
		t.Fatal("reset erased queue or failed")
	}
}
func TestSettingsAndProfilePersistWithoutSecrets(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	v := model.DefaultSettings()
	v.Concurrency = 3
	if e = s.SaveSettings(v); e != nil {
		t.Fatal(e)
	}
	if e = s.PutProfile(model.Profile{ID: "p", Site: "pixiv", Name: "Compte principal", Method: "token"}); e != nil {
		t.Fatal(e)
	}
	state, _ := s.Snapshot()
	if state.Settings.Concurrency != 3 || len(state.Profiles) != 1 {
		t.Fatal("settings/profile not persisted")
	}
}
