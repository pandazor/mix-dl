package downloader

import (
	"dl-manager/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestPartialResumeCountsOnlyNewBytes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "image.jpg")
	os.WriteFile(p+".part", make([]byte, 40), 0600)
	var n int64
	tr := &fileTracker{emit: func(e model.Event) {
		if e.Kind == "bytes" {
			n += e.Bytes
		}
	}}
	tr.begin(p)
	os.WriteFile(p+".part", make([]byte, 100), 0600)
	tr.poll()
	os.Rename(p+".part", p)
	tr.finish(p)
	if n != 60 {
		t.Fatalf("counted %d, want 60", n)
	}
}
func TestYouTubeResumeSeparateStreamBaselines(t *testing.T) {
	var n int64
	tr := &fileTracker{baselines: map[string]int64{"clip.f137.mp4": 80}, emit: func(e model.Event) {
		if e.Kind == "bytes" {
			n += e.Bytes
		}
	}}
	tr.begin("clip.mp4")
	tr.progress("clip.f137.mp4", 100)
	tr.progress("clip.f140.m4a", 30)
	tr.progress("clip.f137.mp4", 110)
	if n != 60 {
		t.Fatalf("counted %d, want 60", n)
	}
}
