package backup

import (
	"bytes"
	"testing"
)

const testPhrase = "une phrase de test assez longue"

func TestEncryptionRoundTripIntegrityAndRandomness(t *testing.T) {
	plain := []byte("secret-fixture-not-for-production")
	one, e := Encrypt(plain, testPhrase)
	if e != nil {
		t.Fatal(e)
	}
	two, e := Encrypt(plain, testPhrase)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(one, plain) || bytes.Equal(one, two) {
		t.Fatal("plaintext or deterministic ciphertext")
	}
	got, e := Decrypt(one, testPhrase)
	if e != nil || !bytes.Equal(got, plain) {
		t.Fatal("roundtrip", e)
	}
	if _, e = Decrypt(one, "autre phrase de test incorrecte"); e == nil {
		t.Fatal("wrong phrase accepted")
	}
	for _, index := range []int{0, 10, 30, len(one) - 1} {
		tampered := bytes.Clone(one)
		tampered[index] ^= 1
		if _, e = Decrypt(tampered, testPhrase); e == nil {
			t.Fatalf("tampered byte %d accepted", index)
		}
	}
}
func TestEncryptionRejectsWeakOrOversizedInputs(t *testing.T) {
	for _, phrase := range []string{"short", "                ", string(bytes.Repeat([]byte("a"), 1025))} {
		if _, e := Encrypt([]byte("x"), phrase); e == nil {
			t.Fatal("invalid phrase accepted")
		}
	}
	if _, e := Encrypt([]byte("x"), "éééééééééééééééé"); e != nil {
		t.Fatal("unicode phrase rejected", e)
	}
	for _, blob := range [][]byte{nil, []byte("DLMBK001"), make([]byte, 32*1024*1024+1)} {
		if _, e := Decrypt(blob, testPhrase); e == nil {
			t.Fatal("malformed/oversized accepted")
		}
	}
	if _, e := Encrypt(make([]byte, 32*1024*1024), testPhrase); e == nil {
		t.Fatal("oversized plaintext accepted")
	}
}
