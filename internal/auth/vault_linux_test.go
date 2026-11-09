package auth

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxVaultPersistsWithoutWritingPlaintext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials")
	keyring := map[string][]byte{}
	command := func(op, id string, input []byte) ([]byte, error) {
		if op == "store" {
			keyring[id] = bytes.Clone(input)
			return nil, nil
		}
		if key := keyring[id]; key != nil {
			return bytes.Clone(key), nil
		}
		return nil, errors.New("entry missing")
	}
	newVault := func() *Vault {
		v, err := New(path)
		if err != nil {
			t.Fatal(err)
		}
		v.keyring = command
		return v
	}
	v := newVault()
	value := []byte("test-only-cookie-and-cache-value")
	if err := v.Save("profile1", value); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(path, "profile1.bin"))
	if err != nil || bytes.Contains(b, value) || len(b) <= len(value) {
		t.Fatalf("secret was not encrypted: %v", err)
	}
	v = newVault()
	got, err := v.Load("profile1")
	if err != nil || !bytes.Equal(got, value) {
		t.Fatalf("reopened vault could not load secret: %v", err)
	}
	if err := v.Delete("profile1"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Load("profile1"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted secret: %v", err)
	}
}

func TestLinuxVaultUnavailableDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials")
	v, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	v.keyring = func(op, id string, input []byte) ([]byte, error) { return nil, errors.New("locked") }
	if err := v.Save("profile1", []byte("synthetic")); err == nil {
		t.Fatal("saved without a keyring")
	}
	if _, err := os.Stat(filepath.Join(path, "profile1.bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unprotected secret file created: %v", err)
	}
	var stored []byte
	v.keyring = func(op, id string, input []byte) ([]byte, error) {
		if op == "store" {
			stored = bytes.Clone(input)
		}
		return bytes.Clone(stored), nil
	}
	if err := v.Save("profile1", []byte("synthetic")); err != nil {
		t.Fatal(err)
	}
	v.keyring = func(op, id string, input []byte) ([]byte, error) { return nil, errors.New("locked") }
	if _, err := v.Load("profile1"); err == nil || !strings.Contains(err.Error(), "Secret Service") {
		t.Fatalf("locked vault error: %v", err)
	}
	if err := v.Save("profile1", []byte("replacement")); err == nil {
		t.Fatal("overwrote protected secret with locked keyring")
	}
	v.keyring = func(op, id string, input []byte) ([]byte, error) { return bytes.Clone(stored), nil }
	got, err := v.Load("profile1")
	if err != nil || string(got) != "synthetic" {
		t.Fatalf("original secret lost: %v", err)
	}
}
