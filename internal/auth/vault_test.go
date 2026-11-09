package auth

import "testing"

func TestCookiesValidation(t *testing.T) {
	if e := ValidateCookies("# Netscape HTTP Cookie File\n.instagram.com\tTRUE\t/\tTRUE\t0\tsessionid\tdummy\n", "instagram"); e != nil {
		t.Fatal(e)
	}
	for _, txt := range []string{"not a cookie file", "# Netscape HTTP Cookie File\n.other.com\tTRUE\t/\tTRUE\t0\tsessionid\tdummy\n"} {
		if ValidateCookies(txt, "instagram") == nil {
			t.Fatal("invalid cookies accepted")
		}
	}
}
