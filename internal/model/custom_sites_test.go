package model

import "testing"

const customA = "custom_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const customB = "custom_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestCustomSiteResolutionAndValidation(t *testing.T) {
	s := DefaultSettings()
	s.CustomSites = map[string]CustomSite{customA: {"Images", "example.org"}, customB: {"Détail", "art.example.org"}}
	if e := s.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, c := range []struct{ url, want string }{{"https://example.org/x", customA}, {"https://cdn.art.example.org/x", customB}, {"https://example.org.evil.net/x", "generic"}, {"https://notexample.org/x", "generic"}, {"https://www.tumblr.com/name", "tumblr"}} {
		got, e := s.ResolveSite(c.url)
		if e != nil || got != c.want {
			t.Fatalf("%s -> %s %v", c.url, got, e)
		}
	}
	for _, domain := range []string{"com", "co.uk", "github.io", "instagram.com", "www.tumblr.com", "https://example.org", "user@example.org", "example.org/path", "example.org:8080", "127.0.0.1", "bad..example.org", "-bad.example.org"} {
		if _, e := NormalizeDomain(domain); e == nil {
			t.Errorf("unsafe domain accepted: %s", domain)
		}
	}
	if d, e := NormalizeDomain(" ART.EXAMPLE.ORG "); e != nil || d != "art.example.org" {
		t.Fatal("normalization", d, e)
	}
	s.CustomSites[customB] = CustomSite{"Other", "example.org"}
	if e := s.Validate(); e == nil {
		t.Fatal("duplicate domain accepted")
	}
}
