package storage

import (
	"dl-manager/internal/model"
	"path/filepath"
	"testing"
)

func TestProgressAtomicallyPersistsUsageAndJob(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "s.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	j := model.Job{ID: "job", Status: "running", Bytes: 51}
	if e = s.Progress(j, 51); e != nil {
		t.Fatal(e)
	}
	v, _ := s.Snapshot()
	if v.UsedBytes != 51 || v.Jobs[0].Bytes != 51 {
		t.Fatalf("inconsistent progress %+v", v)
	}
}
