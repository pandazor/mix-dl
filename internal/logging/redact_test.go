package logging

import (
	"strings"
	"testing"
)

func TestRedactCredentialsAndSignedURLs(t *testing.T) {
	r := NewRedactor([]string{"unique-secret-456"})
	s := r.Redact("failure unique-secret-456 https://host.test/f?token=abc&key=def Authorization: Bearer abcd")
	for _, secret := range []string{"unique-secret-456", "token=abc", "key=def", "Bearer abcd"} {
		if strings.Contains(s, secret) {
			t.Errorf("leaked %s in %s", secret, s)
		}
	}
}
