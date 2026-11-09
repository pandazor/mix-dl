package web

import (
	"bytes"
	"dl-manager/internal/backup"
	"dl-manager/internal/downloader"
	"dl-manager/internal/model"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const backupPhrase = "ma phrase de test de seize caractères"
const backupSecret = "synthetic-api-secret-never-log"

func backupRequest(t *testing.T, a *API, route string, body map[string]any) (int, string) {
	t.Helper()
	b, e := json.Marshal(body)
	if e != nil {
		t.Fatal(e)
	}
	w := request(t, a, "POST", route, string(b))
	return w.Code, w.Body.String()
}
func exportFixture(t *testing.T) (string, *API) {
	t.Helper()
	a, _ := profileAPI(t)
	w := request(t, a, "POST", "profile", `{"name":"Compte test","site":"rule34","method":"api","userId":"123","token":"`+backupSecret+`"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s, _ := a.Store.Snapshot()
	site := s.Settings.Sites["rule34"]
	site.Layout = "flat"
	s.Settings.Sites["rule34"] = site
	a.Manager.SaveSettings(s.Settings)
	p := filepath.Join(t.TempDir(), "sites.dlmbackup")
	code, body := backupRequest(t, a, "backup/export", map[string]any{"path": p, "passphrase": backupPhrase, "confirmation": backupPhrase, "acknowledge": true})
	if code != 200 {
		t.Fatal(code, body)
	}
	if strings.Contains(body, backupSecret) || strings.Contains(body, backupPhrase) {
		t.Fatal("secret response")
	}
	return p, a
}
func TestBackupRoundTripPreservesExistingSettings(t *testing.T) {
	p, source := exportFixture(t)
	encrypted, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(encrypted, []byte(backupSecret)) {
		t.Fatal("plaintext backup")
	}
	a, vault := profileAPI(t)
	state, _ := a.Store.Snapshot()
	global := state.Settings.Destination
	state.Settings.Sites["rule34"] = model.SiteSettings{Layout: "site"}
	if e = a.Manager.SaveSettings(state.Settings); e != nil {
		t.Fatal(e)
	}
	code, body := backupRequest(t, a, "backup/import", map[string]any{"path": p, "passphrase": backupPhrase})
	if code != 200 {
		t.Fatal(code, body)
	}
	got, _ := a.Store.Snapshot()
	original, _ := source.Store.Snapshot()
	if len(got.Profiles) != 1 || got.Profiles[0].ID == original.Profiles[0].ID {
		t.Fatal("missing account/new identity")
	}
	if got.Settings.Destination != global || got.Settings.Sites["rule34"].Layout != "site" || got.Running || len(got.Jobs) != 0 {
		t.Fatal("import altered unrelated state")
	}
	secret, e := vault.Load(got.Profiles[0].ID)
	if e != nil || !bytes.Contains(secret, []byte(backupSecret)) {
		t.Fatal("imported credentials lost", e)
	}
	code, body = backupRequest(t, a, "backup/import", map[string]any{"path": p, "passphrase": backupPhrase, "replaceSites": true})
	if code != 200 {
		t.Fatal(body)
	}
	got, _ = a.Store.Snapshot()
	if len(got.Profiles) != 2 || got.Settings.Sites["rule34"].Layout != "flat" || got.Settings.Sites["rule34"].ProfileID == "" {
		t.Fatal("replace/remap failed")
	}
}
func TestBackupRefusesWrongPhraseCorruptionAndOverwrite(t *testing.T) {
	p, a := exportFixture(t)
	before, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	for _, req := range []map[string]any{{"path": p, "passphrase": backupPhrase, "confirmation": backupPhrase, "acknowledge": true}, {"path": p + ".new.dlmbackup", "passphrase": backupPhrase, "confirmation": "different", "acknowledge": true}, {"path": p + ".new.dlmbackup", "passphrase": backupPhrase, "confirmation": backupPhrase}} {
		if code, _ := backupRequest(t, a, "backup/export", req); code == 200 {
			t.Fatal("unsafe export accepted")
		}
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("existing backup overwritten")
	}
	target, v := profileAPI(t)
	for _, bad := range []bool{false, true} {
		phrase := "incorrect phrase of enough length"
		if bad {
			phrase = backupPhrase
			corrupt := bytes.Clone(before)
			corrupt[len(corrupt)-1] ^= 1
			if e = os.WriteFile(p, corrupt, 0600); e != nil {
				t.Fatal(e)
			}
		}
		code, body := backupRequest(t, target, "backup/import", map[string]any{"path": p, "passphrase": phrase})
		if code == 200 {
			t.Fatal("invalid import accepted")
		}
		if strings.Contains(body, phrase) || strings.Contains(body, backupSecret) {
			t.Fatal("secret in error")
		}
		s, _ := target.Store.Snapshot()
		if len(s.Profiles) != 0 || len(v.values) != 0 {
			t.Fatal("failed import mutated state")
		}
	}
}

type failingBackupVault struct {
	memoryVault
	saves int
}

func (v *failingBackupVault) Save(k string, b []byte) error {
	v.saves++
	if v.saves == 2 {
		return errors.New("synthetic vault failure")
	}
	return v.memoryVault.Save(k, b)
}
func TestBackupRollbackAndActiveQueueProtection(t *testing.T) {
	p, source := exportFixture(t)
	w := request(t, source, "POST", "profile", `{"name":"Second test","site":"rule34","method":"api","userId":"456","token":"second-fixture"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	p += ".second.dlmbackup"
	code, body := backupRequest(t, source, "backup/export", map[string]any{"path": p, "passphrase": backupPhrase, "confirmation": backupPhrase, "acknowledge": true})
	if code != 200 {
		t.Fatal(body)
	}
	target, _ := profileAPI(t)
	v := &failingBackupVault{memoryVault: memoryVault{values: map[string][]byte{}}}
	target.Vault = v
	code, body = backupRequest(t, target, "backup/import", map[string]any{"path": p, "passphrase": backupPhrase})
	if code == 200 {
		t.Fatal("coffre failure accepted")
	}
	state, _ := target.Store.Snapshot()
	if len(state.Profiles) > 0 || len(v.values) > 0 {
		t.Fatal("partial import survived")
	}
	target.Manager.Start()
	code, _ = backupRequest(t, target, "backup/import", map[string]any{"path": p, "passphrase": backupPhrase})
	if code == 200 {
		t.Fatal("active queue import accepted")
	}
	target.Manager.Pause()
}

func TestBackupValidatesPayloadBeforeWritingAndResetsForeignPaths(t *testing.T) {
	p, _ := exportFixture(t)
	raw, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	plain, e := backup.Decrypt(raw, backupPhrase)
	if e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"version", "traversal", "duplicate", "missing-default", "unknown-site", "method", "filter", "unknown-field"} {
		t.Run(kind, func(t *testing.T) {
			var doc backupDocument
			if e = json.Unmarshal(plain, &doc); e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "version":
				doc.Version = 2
			case "traversal":
				doc.Accounts[0].Profile.ID = "../../escape"
			case "duplicate":
				doc.Accounts = append(doc.Accounts, doc.Accounts[0])
			case "missing-default":
				sc := doc.Sites["rule34"]
				sc.ProfileID = "absent"
				doc.Sites["rule34"] = sc
			case "unknown-site":
				doc.Sites["unknown"] = model.SiteSettings{}
			case "method":
				doc.Accounts[0].Credentials.Method = "other"
			case "filter":
				sc := doc.Sites["rule34"]
				sc.Filters = &model.Filters{Images: true, Extensions: []string{"not-valid"}}
				doc.Sites["rule34"] = sc
			}
			b, _ := json.Marshal(doc)
			if kind == "unknown-field" {
				b = []byte("{\"synthetic-secret-field\":true}")
			}
			enc, e := backup.Encrypt(b, backupPhrase)
			if e != nil {
				t.Fatal(e)
			}
			bad := filepath.Join(t.TempDir(), "bad.dlmbackup")
			if e = os.WriteFile(bad, enc, 0600); e != nil {
				t.Fatal(e)
			}
			target, v := profileAPI(t)
			code, body := backupRequest(t, target, "backup/import", map[string]any{"path": bad, "passphrase": backupPhrase})
			if code == 200 {
				t.Fatal("invalid payload accepted")
			}
			if strings.Contains(body, "synthetic-secret-field") {
				t.Fatal("decoder leaked input")
			}
			st, _ := target.Store.Snapshot()
			if len(st.Profiles) > 0 || len(v.values) > 0 {
				t.Fatal("invalid payload partially applied")
			}
		})
	}
	var doc backupDocument
	json.Unmarshal(plain, &doc)
	sc := doc.Sites["rule34"]
	sc.Destination = "/home/old/downloads"
	if runtime.GOOS != "windows" {
		sc.Destination = "C:\\Users\\old\\downloads"
	}
	doc.Sites["rule34"] = sc
	b, _ := json.Marshal(doc)
	enc, e := backup.Encrypt(b, backupPhrase)
	if e != nil {
		t.Fatal(e)
	}
	foreign := filepath.Join(t.TempDir(), "foreign.dlmbackup")
	os.WriteFile(foreign, enc, 0600)
	target, _ := profileAPI(t)
	code, body := backupRequest(t, target, "backup/import", map[string]any{"path": foreign, "passphrase": backupPhrase})
	if code != 200 {
		t.Fatal(body)
	}
	st, _ := target.Store.Snapshot()
	if st.Settings.Sites["rule34"].Destination != "" || !strings.Contains(body, `"resetPaths":1`) {
		t.Fatal("foreign path was not reset")
	}
}
func TestBackupPreservesCookiesAndOAuthSecrets(t *testing.T) {
	source, v := profileAPI(t)
	entries := []backupAccount{
		{model.Profile{ID: "cookie-source", Site: "youtube", Name: "Cookies", Method: "cookies"}, downloader.Credentials{Method: "cookies", Cookies: "# Netscape HTTP Cookie File\n.youtube.com\tTRUE\t/\tTRUE\t0\tSID\tsynthetic-cookie\n"}},
		{model.Profile{ID: "oauth-source", Site: "tumblr", Name: "OAuth", Method: "oauth1"}, downloader.Credentials{Method: "oauth1", Token: "synthetic-token", TokenSecret: "synthetic-secret"}},
	}
	for _, entry := range entries {
		b, _ := json.Marshal(entry.Credentials)
		v.Save(entry.Profile.ID, b)
		if e := source.Store.PutProfileWithDefault(entry.Profile, true); e != nil {
			t.Fatal(e)
		}
	}
	p := filepath.Join(t.TempDir(), "types.dlmbackup")
	code, body := backupRequest(t, source, "backup/export", map[string]any{"path": p, "passphrase": backupPhrase, "confirmation": backupPhrase, "acknowledge": true})
	if code != 200 {
		t.Fatal(body)
	}
	target, tv := profileAPI(t)
	code, body = backupRequest(t, target, "backup/import", map[string]any{"path": p, "passphrase": backupPhrase})
	if code != 200 {
		t.Fatal(body)
	}
	st, _ := target.Store.Snapshot()
	if len(st.Profiles) != 2 {
		t.Fatal("accounts lost")
	}
	for _, p := range st.Profiles {
		var got downloader.Credentials
		json.Unmarshal(tv.values[p.ID], &got)
		for _, entry := range entries {
			if p.Site == entry.Profile.Site && got != entry.Credentials {
				t.Fatal("credentials changed")
			}
		}
	}
}
