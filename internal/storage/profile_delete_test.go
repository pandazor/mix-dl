package storage

import (
	"dl-manager/internal/model"
	"path/filepath"
	"testing"
)

func TestDeletingDefaultProfileClearsSiteReference(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	cfg := model.DefaultSettings()
	cfg.Sites["artstation"] = model.SiteSettings{ProfileID: "removed", Destination: t.TempDir()}
	cfg.Sites["youtube"] = model.SiteSettings{ProfileID: "retained"}
	if e = s.SaveSettings(cfg); e != nil {
		t.Fatal(e)
	}
	if e = s.PutProfile(model.Profile{ID: "removed", Site: "artstation"}); e != nil {
		t.Fatal(e)
	}
	if e = s.DeleteProfile("removed"); e != nil {
		t.Fatal(e)
	}
	got, e := s.Snapshot()
	if e != nil {
		t.Fatal(e)
	}
	if got.Settings.Sites["artstation"].ProfileID != "" {
		t.Fatal("deleted account still selected by default")
	}
	if got.Settings.Sites["artstation"].Destination != cfg.Sites["artstation"].Destination || got.Settings.Sites["youtube"].ProfileID != "retained" {
		t.Fatal("unrelated settings changed")
	}
}
