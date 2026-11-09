package web

import (
	"dl-manager/internal/model"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createCustom(t *testing.T, a *API, name, domain string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name, "domain": domain})
	w := request(t, a, "POST", "custom-site", string(body))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var r struct {
		ID string `json:"id"`
	}
	json.Unmarshal(w.Body.Bytes(), &r)
	if r.ID == "" {
		t.Fatal("missing ID")
	}
	return r.ID
}
func TestCustomSiteProfileInheritanceRenameAndDeletion(t *testing.T) {
	a, _ := profileAPI(t)
	defer a.Close()
	id := createCustom(t, a, "Images", "ART.EXAMPLE.ORG")
	other := createCustom(t, a, "Autre", "images.example.net")
	s, _ := a.Store.Snapshot()
	if s.Settings.CustomSites[id].Domain != "art.example.org" {
		t.Fatal("not persisted")
	}
	c := s.Settings.Sites[id]
	c.Destination = filepath.Join(a.DataDir, "custom")
	s.Settings.Sites[id] = c
	if e := a.Manager.SaveSettings(s.Settings); e != nil {
		t.Fatal(e)
	}
	cookie := filepath.Join(a.DataDir, "cookies.txt")
	os.WriteFile(cookie, []byte("# Netscape HTTP Cookie File\n.art.example.org\tTRUE\t/\tTRUE\t2147483647\tsession\ttest\n"), 0600)
	body, _ := json.Marshal(map[string]any{"name": "Compte", "site": id, "method": "cookies", "cookiePath": cookie})
	w := request(t, a, "POST", "profile", string(body))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = request(t, a, "POST", "jobs", `{"urls":["https://art.example.org/artist","https://images.example.net/artist"]}`)
	if !strings.Contains(w.Body.String(), `"added":2`) {
		t.Fatal(w.Body.String())
	}
	s, _ = a.Store.Snapshot()
	if len(s.Jobs) != 2 || s.Jobs[0].Site != id || s.Jobs[0].Destination != c.Destination || s.Jobs[1].Site != other || s.Jobs[1].Destination != a.DataDir {
		t.Fatal("incorrect routing/destination")
	}
	p, e := model.ResolveProfile(s.Jobs[0], s)
	if e != nil || p != s.Profiles[0].ID {
		t.Fatal("default account not inherited")
	}
	if p, e = model.ResolveProfile(s.Jobs[1], s); e != nil || p != "" {
		t.Fatal("account crossed site")
	}
	for _, domain := range []string{"art.example.org", "different.example.org"} {
		body, _ := json.Marshal(map[string]string{"id": id, "name": "Renommé", "domain": domain})
		w = request(t, a, "POST", "custom-site", string(body))
		if (domain == "art.example.org") != (w.Code == 200) {
			t.Fatal("rename/domain guard", w.Body.String())
		}
	}
	w = request(t, a, "POST", "custom-site/delete", `{"id":"`+id+`"}`)
	if w.Code == 200 {
		t.Fatal("used site deleted")
	}
	s, _ = a.Store.Snapshot()
	delete(s.Settings.CustomSites, id)
	if e = a.Manager.SaveSettings(s.Settings); e == nil {
		t.Fatal("settings bypass removed used site")
	}
	w = request(t, a, "GET", "state", "")
	if !strings.Contains(w.Body.String(), "Renommé") {
		t.Fatal("site missing from catalogue")
	}
	empty := createCustom(t, a, "Libre", "free.example.net")
	w = request(t, a, "POST", "custom-site/delete", `{"id":"`+empty+`"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
}
func TestCustomCookieScopeAndDuplicateDomain(t *testing.T) {
	a, _ := profileAPI(t)
	id := createCustom(t, a, "Images", "art.example.org")
	w := request(t, a, "POST", "custom-site", `{"name":"doublon","domain":"ART.EXAMPLE.ORG"}`)
	if w.Code == 200 {
		t.Fatal("duplicate")
	}
	for _, host := range []string{".example.org", ".evilart.example.org", ".art.example.org.evil.net", ".co.uk"} {
		p := filepath.Join(a.DataDir, "cookies.txt")
		os.WriteFile(p, []byte(host+"\tTRUE\t/\tTRUE\t2147483647\tsession\tsynthetic\n"), 0600)
		b, _ := json.Marshal(map[string]string{"name": "Wrong", "site": id, "method": "cookies", "cookiePath": p})
		w = request(t, a, "POST", "profile", string(b))
		if w.Code == 200 {
			t.Fatal("wrong cookie domain accepted", host)
		}
	}
}
func TestCustomSitesEncryptedBackupRemapsAndPreservesExisting(t *testing.T) {
	a, _ := profileAPI(t)
	id := createCustom(t, a, "Source", "art.example.org")
	cookie := filepath.Join(a.DataDir, "cookies.txt")
	os.WriteFile(cookie, []byte(".art.example.org\tTRUE\t/\tTRUE\t2147483647\tsession\tBACKUP_SYNTHETIC\n"), 0600)
	b, _ := json.Marshal(map[string]string{"name": "Compte", "site": id, "method": "cookies", "cookiePath": cookie})
	if w := request(t, a, "POST", "profile", string(b)); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	path := filepath.Join(a.DataDir, "custom.dlmbackup")
	b, _ = json.Marshal(map[string]any{"path": path, "passphrase": "phrase for custom backup", "confirmation": "phrase for custom backup", "acknowledge": true})
	if w := request(t, a, "POST", "backup/export", string(b)); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "BACKUP_SYNTHETIC") {
		t.Fatal("cleartext export")
	}
	target, _ := profileAPI(t)
	existing := createCustom(t, target, "Nom local", "art.example.org")
	s, _ := target.Store.Snapshot()
	cfg := s.Settings.Sites[existing]
	cfg.Destination = filepath.Join(target.DataDir, "keep")
	s.Settings.Sites[existing] = cfg
	target.Manager.SaveSettings(s.Settings)
	b, _ = json.Marshal(map[string]any{"path": path, "passphrase": "phrase for custom backup", "replaceSites": false})
	if w := request(t, target, "POST", "backup/import", string(b)); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s, _ = target.Store.Snapshot()
	if len(s.Settings.CustomSites) != 1 || s.Settings.CustomSites[existing].Name != "Nom local" || s.Settings.Sites[existing].Destination != cfg.Destination || len(s.Profiles) != 1 || s.Profiles[0].Site != existing {
		t.Fatal("merge replaced local state or failed to remap")
	}
	firstImported := s.Profiles[0].ID
	b, _ = json.Marshal(map[string]any{"path": path, "passphrase": "phrase for custom backup", "replaceSites": true})
	if w := request(t, target, "POST", "backup/import", string(b)); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s, _ = target.Store.Snapshot()
	if s.Settings.CustomSites[existing].Name != "Source" || s.Settings.Sites[existing].ProfileID == "" || s.Settings.Sites[existing].ProfileID == firstImported {
		t.Fatal("explicit replacement failed")
	}
	fresh, _ := profileAPI(t)
	if w := request(t, fresh, "POST", "backup/import", string(b)); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s, _ = fresh.Store.Snapshot()
	if len(s.Settings.CustomSites) != 1 || len(s.Profiles) != 1 || s.Settings.CustomSites[s.Profiles[0].Site].Domain != "art.example.org" {
		t.Fatal("new installation import failed")
	}
}
func TestImportKeepsCustomSiteWithInheritedSettings(t *testing.T) {
	source, _ := profileAPI(t)
	id := createCustom(t, source, "Export", "art.example.org")
	s, _ := source.Store.Snapshot()
	s.Settings.Sites[id] = model.SiteSettings{Destination: filepath.Join(source.DataDir, "remote")}
	source.Manager.SaveSettings(s.Settings)
	path := filepath.Join(source.DataDir, "inherited.dlmbackup")
	b, _ := json.Marshal(map[string]any{"path": path, "passphrase": "long phrase for backup", "confirmation": "long phrase for backup", "acknowledge": true})
	if w := request(t, source, "POST", "backup/export", string(b)); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	target, _ := profileAPI(t)
	local := createCustom(t, target, "Local", "art.example.org")
	b, _ = json.Marshal(map[string]any{"path": path, "passphrase": "long phrase for backup", "replaceSites": false})
	if w := request(t, target, "POST", "backup/import", string(b)); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s, _ = target.Store.Snapshot()
	if s.Settings.Sites[local].Destination != "" || s.Settings.CustomSites[local].Name != "Local" {
		t.Fatal("import replaced implicit inherited settings")
	}
}
