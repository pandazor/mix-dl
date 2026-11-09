package web

import (
	"bytes"
	"dl-manager/internal/auth"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupImportProtectsCredentialsWithDPAPI(t *testing.T) {
	p, _ := exportFixture(t)
	a, _ := profileAPI(t)
	vault, e := auth.New(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	a.Vault = vault
	code, body := backupRequest(t, a, "backup/import", map[string]any{"path": p, "passphrase": backupPhrase})
	if code != 200 {
		t.Fatal(body)
	}
	st, _ := a.Store.Snapshot()
	id := st.Profiles[0].ID
	secret, e := vault.Load(id)
	if e != nil || !bytes.Contains(secret, []byte(backupSecret)) {
		t.Fatal("import lost credentials", e)
	}
	disk, e := os.ReadFile(filepath.Join(vault.Dir, id+".bin"))
	if e != nil || bytes.Contains(disk, []byte(backupSecret)) {
		t.Fatal("plaintext local vault", e)
	}
}
