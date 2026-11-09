package downloader

import (
	"context"
	"dl-manager/internal/auth"
	"dl-manager/internal/model"
	"dl-manager/internal/storage"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWindowsPathStripPreservesOrdinaryNameSuffixes(t *testing.T) {
	j := model.Job{URL: "https://example.org/jeff.jpg", Site: "generic", Destination: t.TempDir(), Filters: model.DefaultFilters()}
	inv, err := BuildInvocation(j, model.DefaultSettings(), Credentials{}, t.TempDir(), "archive", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Extractor map[string]any `json:"extractor"`
	}
	if err = json.Unmarshal(inv.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Extractor["path-strip"] != "windows" {
		t.Fatalf("strip must remove trailing Windows dots/spaces, not f/x/1: %q", cfg.Extractor["path-strip"])
	}
}

func TestDirectLinksWithSameBasenameBothDownload(t *testing.T) {
	tools := integrationTools()
	if _, err := os.Stat(filepath.Join(tools, executableName("gallery-dl"))); err != nil {
		t.Skip("bundled engine absent")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("fixture:" + r.URL.Path))
	}))
	defer srv.Close()
	data := t.TempDir()
	s, err := storage.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v, err := auth.New(filepath.Join(data, "credentials"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &Runner{Store: s, Vault: v, DataDir: data, ToolsDir: tools}
	dest := t.TempDir()
	seen := map[string]bool{}
	for _, path := range []string{"/a/picture.jpg", "/b/picture.jpg"} {
		j := model.Job{ID: model.ID(), URL: srv.URL + path, Site: "generic", Destination: dest, Layout: "flat", Filename: "id-title", Filters: model.DefaultFilters()}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err = runner.Run(ctx, j, func(model.Event) {})
		cancel()
		if err != nil {
			t.Fatal(err)
		}
	}
	filepath.WalkDir(dest, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := os.ReadFile(p)
			seen[string(b)] = true
		}
		return err
	})
	for _, path := range []string{"/a/picture.jpg", "/b/picture.jpg"} {
		if !seen["fixture:"+path] {
			t.Errorf("distinct source lost: %s", path)
		}
	}
}
