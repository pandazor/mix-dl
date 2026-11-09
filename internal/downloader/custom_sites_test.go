package downloader

import (
	"dl-manager/internal/model"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomInvocationUsesStableDomainAndRejectsMismatchedURL(t *testing.T) {
	id := "custom_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	s := model.DefaultSettings()
	s.CustomSites = map[string]model.CustomSite{id: {Name: "Nom modifiable", Domain: "art.example.org"}}
	j := model.Job{Site: id, URL: "https://art.example.org/artist", Destination: t.TempDir(), Layout: "creator", Filters: model.DefaultFilters()}
	inv, e := BuildInvocation(j, s, Credentials{Method: "cookies", Cookies: "synthetic"}, t.TempDir(), filepath.Join(t.TempDir(), "archive"), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	var cfg map[string]any
	json.Unmarshal(inv.Config, &cfg)
	ex := cfg["extractor"].(map[string]any)
	d := ex["directory"].([]any)
	if d[0] != "art.example.org" || strings.Contains(string(inv.Config), id) || ex["cookies"] == nil {
		t.Fatal("wrong custom extractor or directory")
	}
	s.CustomSites[id] = model.CustomSite{Name: "Nouveau nom", Domain: "art.example.org"}
	inv2, e := BuildInvocation(j, s, Credentials{}, t.TempDir(), "a", t.TempDir())
	if e != nil || !strings.Contains(string(inv2.Config), "art.example.org") {
		t.Fatal("rename changed folder")
	}
	j.URL = "https://art.example.org.evil.net/a"
	if _, e = BuildInvocation(j, s, Credentials{Method: "cookies"}, t.TempDir(), "a", t.TempDir()); e == nil {
		t.Fatal("mismatched site accepted")
	}
}
