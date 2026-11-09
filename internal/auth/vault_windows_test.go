package auth

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSecretRoundTripAndNoPlaintext(t *testing.T) {
	if _, e := crypt([]byte("test-only-probe"), true); e != nil && os.Getenv("DLM_RESTRICTED_TESTS") == "1" {
		t.Skipf("DPAPI unavailable in restricted execution profile (also reproduced with .NET): %v", e)
	}
	v, e := New(t.TempDir())
	if e != nil {
		t.Fatalf("private dir: %v", e)
	}
	value := []byte("test-only-credential-0123456789")
	if e = v.Save("profile1", value); e != nil {
		t.Fatalf("save: %v", e)
	}
	disk, e := os.ReadFile(filepath.Join(v.Dir, "profile1.bin"))
	if e != nil {
		t.Fatalf("read: %v", e)
	}
	if bytes.Contains(disk, value) {
		t.Fatal("secret stored plaintext")
	}
	got, e := v.Load("profile1")
	if e != nil || !bytes.Equal(got, value) {
		t.Fatalf("roundtrip %v", e)
	}
	if e = v.Save("../escape", value); e == nil {
		t.Fatal("path traversal accepted")
	}
}
