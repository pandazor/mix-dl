package downloader

import (
	"bytes"
	"context"
	"dl-manager/internal/auth"
	"dl-manager/internal/model"
	"dl-manager/internal/storage"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRealGalleryDownloadResumeAndArchive(t *testing.T) {
	tools, _ := filepath.Abs("../../tools")
	if _, e := os.Stat(filepath.Join(tools, "gallery-dl.exe")); e != nil {
		t.Skip("run scripts/fetch-tools.py for engine integration")
	}
	payload := bytes.Repeat([]byte("test-only-image-content"), 6000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "fixture.jpg", time.Unix(1700000000, 0), bytes.NewReader(payload))
	}))
	defer server.Close()
	data := t.TempDir()
	s, e := storage.Open(filepath.Join(data, "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	v, e := auth.New(filepath.Join(data, "creds"))
	if e != nil {
		t.Fatal(e)
	}
	cfg := model.DefaultSettings()
	cfg.RateKiB = 48
	s.SaveSettings(cfg)
	r := &Runner{Store: s, Vault: v, DataDir: data, ToolsDir: tools}
	dest := t.TempDir()
	j := model.Job{ID: "integration", URL: server.URL + "/fixture.jpg", Site: "generic", Destination: dest, Layout: "flat", Filters: model.DefaultFilters()}
	ctx, cancel := context.WithCancel(context.Background())
	var count int64
	var logs []string
	e = r.Run(ctx, j, func(ev model.Event) {
		if ev.Kind == "bytes" {
			count += ev.Bytes
			if count > 12000 {
				cancel()
			}
		}
		if ev.Kind == "log" {
			logs = append(logs, ev.Message)
		}
	})
	cancel()
	if e == nil || count == 0 {
		t.Fatalf("did not interrupt transfer: %v %d %v", e, count, logs)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel2()
	e = r.Run(ctx2, j, func(ev model.Event) {
		if ev.Kind == "bytes" {
			count += ev.Bytes
		}
		if ev.Kind == "log" {
			logs = append(logs, ev.Message)
		}
	})
	if e != nil {
		t.Fatalf("resume: %v %v", e, logs)
	}
	entries, _ := os.ReadDir(dest)
	var found int
	for _, entry := range entries {
		if !entry.IsDir() && !bytes.HasSuffix([]byte(entry.Name()), []byte(".part")) {
			b, _ := os.ReadFile(filepath.Join(dest, entry.Name()))
			if bytes.Equal(b, payload) {
				found++
			}
		}
	}
	if found != 1 {
		t.Fatalf("expected one intact file; entries=%v logs=%v", entries, logs)
	}
	var replayBytes int64
	ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel3()
	e = r.Run(ctx3, j, func(ev model.Event) {
		if ev.Kind == "bytes" {
			replayBytes += ev.Bytes
		}
	})
	if e != nil || replayBytes != 0 {
		t.Fatalf("archive redownloaded: %v %d", e, replayBytes)
	}
	if count != int64(len(payload)) {
		t.Fatalf("quota count=%d payload=%d", count, len(payload))
	}
}
