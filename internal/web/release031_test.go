package web

import (
	"context"
	"dl-manager/internal/backup"
	"dl-manager/internal/model"
	"dl-manager/internal/queue"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSaveGlobalDestinationPreservesSiteOverridesAndOtherSettings(t *testing.T) {
	a, _ := profileAPI(t)
	before, _ := a.Manager.Snapshot()
	before.Settings.Concurrency = 3
	before.Settings.Sites["rule34"] = model.SiteSettings{Destination: filepath.Join(t.TempDir(), "custom"), Layout: "flat"}
	if e := a.Manager.SaveSettings(before.Settings); e != nil {
		t.Fatal(e)
	}
	dest := filepath.Join(t.TempDir(), "global")
	body, _ := json.Marshal(map[string]string{"destination": dest})
	w := request(t, a, "POST", "settings/destination", string(body))
	if w.Code != 200 {
		t.Fatalf("global save failed: %d %s", w.Code, w.Body.String())
	}
	got, _ := a.Manager.Snapshot()
	want := before.Settings
	want.Destination = dest
	if !reflect.DeepEqual(got.Settings, want) {
		t.Fatal("partial destination save overwrote other settings")
	}
	w = request(t, a, "POST", "jobs", `{"urls":["https://www.instagram.com/test-fixture/"]}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	got, _ = a.Manager.Snapshot()
	if len(got.Jobs) != 1 || got.Jobs[0].Destination != dest {
		t.Fatal("new inherited job ignored global")
	}
	for _, bad := range []string{"relative/path", "", "   "} {
		body, _ = json.Marshal(map[string]string{"destination": bad})
		if request(t, a, "POST", "settings/destination", string(body)).Code == 200 {
			t.Fatal("invalid global accepted")
		}
	}
	got, _ = a.Manager.Snapshot()
	if !reflect.DeepEqual(got.Settings, want) {
		t.Fatal("failed save changed settings")
	}
}

type backupRunningWorker struct{ started chan model.Job }

func (r backupRunningWorker) Run(ctx context.Context, j model.Job, emit func(model.Event)) error {
	r.started <- j
	<-ctx.Done()
	return ctx.Err()
}
func TestExportDuringDownloadAndAfterIndividualPause(t *testing.T) {
	_, a := exportFixture(t)
	a.Manager.Close()
	worker := backupRunningWorker{started: make(chan model.Job, 1)}
	m := queue.New(a.Store, worker)
	a.Manager = m
	t.Cleanup(m.Close)
	w := request(t, a, "POST", "jobs", `{"urls":["https://rule34.xxx/index.php?page=post&s=list&tags=test"]}`)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if e := m.Start(); e != nil {
		t.Fatal(e)
	}
	var j model.Job
	select {
	case j = <-worker.started:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not start")
	}
	for _, paused := range []bool{false, true} {
		if paused {
			if e := m.Action(j.ID, "pause"); e != nil {
				t.Fatal(e)
			}
			deadline := time.Now().Add(3 * time.Second)
			for {
				s, _ := m.Snapshot()
				if s.Active == 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("worker did not stop")
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
		before, _ := m.Snapshot()
		p := filepath.Join(t.TempDir(), "export.dlmbackup")
		code, body := backupRequest(t, a, "backup/export", map[string]any{"path": p, "passphrase": backupPhrase, "confirmation": backupPhrase, "acknowledge": true})
		if code != 200 {
			t.Fatalf("export paused=%v: %d %s", paused, code, body)
		}
		raw, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		plain, e := backup.Decrypt(raw, backupPhrase)
		if e != nil {
			t.Fatal(e)
		}
		var doc backupDocument
		if e = json.Unmarshal(plain, &doc); e != nil {
			t.Fatal(e)
		}
		if len(doc.Accounts) != 1 || doc.Accounts[0].Credentials.Token != backupSecret {
			t.Fatal("backup lost active account")
		}
		after, _ := m.Snapshot()
		if !reflect.DeepEqual(before, after) {
			t.Fatal("export changed queue or settings")
		}
	}
}

type backupReadProbe struct {
	CredentialVault
	entered chan struct{}
	release chan struct{}
	once    bool
}

func (v *backupReadProbe) Load(id string) ([]byte, error) {
	if !v.once {
		v.once = true
		close(v.entered)
		<-v.release
	}
	return v.CredentialVault.Load(id)
}
func TestExportSerializesCredentialEdits(t *testing.T) {
	_, a := exportFixture(t)
	s, _ := a.Store.Snapshot()
	id := s.Profiles[0].ID
	probe := &backupReadProbe{CredentialVault: a.Vault, entered: make(chan struct{}), release: make(chan struct{})}
	a.Vault = probe
	p := filepath.Join(t.TempDir(), "snapshot.dlmbackup")
	exported := make(chan int, 1)
	go func() {
		code, _ := backupRequest(t, a, "backup/export", map[string]any{"path": p, "passphrase": backupPhrase, "confirmation": backupPhrase, "acknowledge": true})
		exported <- code
	}()
	select {
	case <-probe.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("export did not read vault")
	}
	edited := make(chan int, 1)
	body, _ := json.Marshal(map[string]string{"id": id, "name": "Updated", "site": "rule34", "method": "api", "token": "new-synthetic-token"})
	go func() { edited <- request(t, a, "POST", "profile", string(body)).Code }()
	select {
	case <-edited:
		close(probe.release)
		t.Fatal("credential edit overtook snapshot")
	case <-time.After(50 * time.Millisecond):
	}
	close(probe.release)
	if <-exported != 200 || <-edited != 200 {
		t.Fatal("export/edit failed")
	}
	raw, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	plain, e := backup.Decrypt(raw, backupPhrase)
	if e != nil {
		t.Fatal(e)
	}
	var doc backupDocument
	if e = json.Unmarshal(plain, &doc); e != nil {
		t.Fatal(e)
	}
	if len(doc.Accounts) != 1 || doc.Accounts[0].Credentials.Token != backupSecret || doc.Accounts[0].Profile.Name != "Compte test" {
		t.Fatal("inconsistent exported account")
	}
	current, _ := a.Store.Snapshot()
	if current.Profiles[0].Name != "Updated" {
		t.Fatal("concurrent edit lost")
	}
}
