package web

import (
	"context"
	"dl-manager/internal/downloader"
	"dl-manager/internal/model"
	"dl-manager/internal/queue"
	"dl-manager/internal/storage"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type memoryVault struct {
	values map[string][]byte
	fail   bool
}

func (v *memoryVault) Save(k string, b []byte) error {
	if v.fail {
		return errors.New("vault locked")
	}
	v.values[k] = append([]byte(nil), b...)
	return nil
}
func (v *memoryVault) Load(k string) ([]byte, error) {
	b, ok := v.values[k]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), b...), nil
}
func (v *memoryVault) Delete(k string) error { delete(v.values, k); return nil }
func profileAPI(t *testing.T) (*API, *memoryVault) {
	t.Helper()
	d := t.TempDir()
	s, e := storage.Open(filepath.Join(d, "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	v := &memoryVault{values: map[string][]byte{}}
	m := queue.New(s, idleRunner{})
	t.Cleanup(func() { m.Close(); s.Close() })
	cfg := model.DefaultSettings()
	cfg.Destination = d
	if e = m.SaveSettings(cfg); e != nil {
		t.Fatal(e)
	}
	return New(s, m, v, d, d), v
}
func request(t *testing.T, a *API, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "/api/"+path, strings.NewReader(body))
	r.Header.Set("X-DL-Token", a.Token)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	return w
}
func savedProfile(t *testing.T, a *API) model.Profile {
	t.Helper()
	s, e := a.Store.Snapshot()
	if e != nil || len(s.Profiles) != 1 {
		t.Fatalf("profile missing: %v %#v", e, s.Profiles)
	}
	return s.Profiles[0]
}
func TestProfileEditKeepsSecretsAndReadOnlyReturnsPresence(t *testing.T) {
	a, v := profileAPI(t)
	w := request(t, a, "POST", "profile", `{"name":"Principal","site":"rule34","method":"api","userId":"12345","token":"test-secret-never-return"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	p := savedProfile(t, a)
	w = request(t, a, "GET", "profile?id="+p.ID, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"userId":"12345"`) || !strings.Contains(w.Body.String(), `"hasToken":true`) || strings.Contains(w.Body.String(), "test-secret-never-return") {
		t.Fatalf("unsafe or missing readback: %d %s", w.Code, w.Body.String())
	}
	w = request(t, a, "POST", "profile", `{"id":"`+p.ID+`","name":"Renomme","site":"rule34","method":"api","userId":"12345","token":""}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var c downloader.Credentials
	json.Unmarshal(v.values[p.ID], &c)
	if c.Token != "test-secret-never-return" {
		t.Fatal("blank field discarded existing token")
	}
	if savedProfile(t, a).Name != "Renomme" {
		t.Fatal("rename missing")
	}
	s, _ := a.Store.Snapshot()
	if s.Settings.Sites["rule34"].ProfileID != p.ID {
		t.Fatal("first account not selected as default")
	}
	// New application facade over persisted state/vault must return the same safe details.
	b := New(a.Store, a.Manager, v, a.DataDir, a.ToolsDir)
	w = request(t, b, "GET", "profile?id="+p.ID, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"hasToken":true`) {
		t.Fatal("reopen lost account")
	}
}
func TestProfileVaultFailureDoesNotReportSuccess(t *testing.T) {
	a, v := profileAPI(t)
	v.fail = true
	w := request(t, a, "POST", "profile", `{"name":"Principal","site":"rule34","method":"api","userId":"12345","token":"dummy"}`)
	if w.Code == 200 {
		t.Fatal("false success")
	}
	s, _ := a.Store.Snapshot()
	if len(s.Profiles) != 0 || s.Settings.Sites["rule34"].ProfileID != "" {
		t.Fatal("failed secret save published metadata")
	}
}
func TestProfileEditRejectsUnknownIDAndMethodChangeWithoutSecret(t *testing.T) {
	a, _ := profileAPI(t)
	w := request(t, a, "POST", "profile", `{"id":"missing","name":"X","site":"rule34","method":"api","userId":"1","token":"dummy"}`)
	if w.Code == 200 {
		t.Fatal("unknown edit created account")
	}
	w = request(t, a, "POST", "profile", `{"name":"X","site":"rule34","method":"api","userId":"1","token":"dummy"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	p := savedProfile(t, a)
	w = request(t, a, "POST", "profile", `{"id":"`+p.ID+`","name":"X","site":"rule34","method":"cookies"}`)
	if w.Code == 200 {
		t.Fatal("method change reused incompatible token")
	}
}
func TestSecondProfileDoesNotReplaceDefault(t *testing.T) {
	a, _ := profileAPI(t)
	body := `{"name":"X","site":"rule34","method":"api","userId":"1","token":"dummy"}`
	request(t, a, "POST", "profile", body)
	p := savedProfile(t, a)
	w := request(t, a, "POST", "profile", body)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s, _ := a.Store.Snapshot()
	if len(s.Profiles) != 2 || s.Settings.Sites["rule34"].ProfileID != p.ID {
		t.Fatal("second account stole default")
	}
}

type captureRunner struct{ jobs chan model.Job }

func (r captureRunner) Run(_ context.Context, j model.Job, _ func(model.Event)) error {
	r.jobs <- j
	return nil
}
func TestSavedRule34AccountReachesExistingTaskAfterRestart(t *testing.T) {
	a, v := profileAPI(t)
	w := request(t, a, "POST", "jobs", `{"urls":["https://rule34.xxx/index.php?page=post&s=list&tags=landscape"]}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = request(t, a, "POST", "profile", `{"name":"Principal","site":"rule34","method":"api","userId":"12345","token":"fixture-secret"}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	p := savedProfile(t, a)
	a.Manager.Close()
	a.Store.Close()
	store, e := storage.Open(filepath.Join(a.DataDir, "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	r := captureRunner{jobs: make(chan model.Job, 1)}
	m := queue.New(store, r)
	defer m.Close()
	if e = m.Start(); e != nil {
		t.Fatal(e)
	}
	select {
	case j := <-r.jobs:
		if j.ProfileID != p.ID {
			t.Fatal("saved account not inherited")
		}
		b, e := v.Load(j.ProfileID)
		if e != nil {
			t.Fatal(e)
		}
		var c downloader.Credentials
		if e = json.Unmarshal(b, &c); e != nil {
			t.Fatal(e)
		}
		inv, e := downloader.BuildInvocation(j, model.DefaultSettings(), c, t.TempDir(), "archive.db", t.TempDir())
		if e != nil {
			t.Fatal(e)
		}
		var config map[string]any
		if e = json.Unmarshal(inv.Config, &config); e != nil {
			t.Fatal(e)
		}
		ext := config["extractor"].(map[string]any)["rule34"].(map[string]any)
		if ext["api-key"] != "fixture-secret" || ext["user-id"] != "12345" {
			t.Fatal("wrong Rule34 engine configuration")
		}
		if strings.Contains(strings.Join(inv.Args, " "), "fixture-secret") {
			t.Fatal("token on process command line")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("task not started")
	}
}
func TestCookieEditKeepsImportedCookies(t *testing.T) {
	a, v := profileAPI(t)
	file := filepath.Join(t.TempDir(), "cookies.txt")
	if e := os.WriteFile(file, []byte(".instagram.com\tTRUE\t/\tTRUE\t0\tsessionid\tsynthetic-cookie"), 0600); e != nil {
		t.Fatal(e)
	}
	body, _ := json.Marshal(map[string]any{"name": "Cookies", "site": "instagram", "method": "cookies", "cookiePath": file})
	w := request(t, a, "POST", "profile", string(body))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	p := savedProfile(t, a)
	before := string(v.values[p.ID])
	w = request(t, a, "POST", "profile", `{"id":"`+p.ID+`","name":"Renamed","site":"instagram","method":"cookies"}`)
	if w.Code != 200 || string(v.values[p.ID]) != before {
		t.Fatal("blank edit lost imported cookies", w.Body.String())
	}
}

func TestProfileDefaultCheckboxHonorsFalse(t *testing.T) {
	a, _ := profileAPI(t)
	w := request(t, a, "POST", "profile", `{"name":"X","site":"rule34","method":"api","userId":"1","token":"dummy","setDefault":false}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	p := savedProfile(t, a)
	s, _ := a.Store.Snapshot()
	if s.Settings.Sites["rule34"].ProfileID != "" {
		t.Fatal("unchecked new account became default")
	}
	for _, choice := range []string{"true", "false"} {
		w = request(t, a, "POST", "profile", `{"id":"`+p.ID+`","name":"X","site":"rule34","method":"api","setDefault":`+choice+`}`)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		s, _ = a.Store.Snapshot()
		want := ""
		if choice == "true" {
			want = p.ID
		}
		if s.Settings.Sites["rule34"].ProfileID != want {
			t.Fatal("default toggle ignored", choice)
		}
	}
}
