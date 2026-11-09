package storage

import (
	"dl-manager/internal/model"
	"path/filepath"
	"testing"
)

func TestImportSitesRollsBackAccountsAndSettingsTogether(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	cfg := model.DefaultSettings()
	cfg.Destination = t.TempDir()
	if e = s.SaveSettings(cfg); e != nil {
		t.Fatal(e)
	}
	existing := model.Profile{ID: "existing", Site: "rule34", Name: "Original"}
	if e = s.PutProfile(existing); e != nil {
		t.Fatal(e)
	}
	changed := cfg
	changed.Destination = t.TempDir()
	e = s.ImportSites(changed, []model.Profile{{ID: "new", Name: "New"}, existing})
	if e == nil {
		t.Fatal("duplicate insert accepted")
	}
	state, e := s.Snapshot()
	if e != nil {
		t.Fatal(e)
	}
	if len(state.Profiles) != 1 || state.Profiles[0].ID != "existing" || state.Settings.Destination != cfg.Destination {
		t.Fatal("partial transaction persisted")
	}
}
