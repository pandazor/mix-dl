package model

import "testing"

func TestValidateURL(t *testing.T) {
	for _, u := range []string{"file:///C:/secret", "--exec=bad", "https://user:password@example.com/", "https://example.com/\n--exec=bad", "javascript:alert(1)"} {
		if _, err := ParseURL(u); err == nil {
			t.Errorf("accepted unsafe URL %q", u)
		}
	}
	p, err := ParseURL("https://www.youtube.com/watch?v=abc")
	if err != nil || p != "youtube" {
		t.Fatalf("%s %v", p, err)
	}
	p, err = ParseURL("https://example.org/gallery/1")
	if err != nil || p != "generic" {
		t.Fatalf("%s %v", p, err)
	}
}
func TestFiltersRejectCodeAndEmptySelection(t *testing.T) {
	f := Filters{Images: true, Extensions: []string{"jpg", "png' or True or '"}}
	if f.Validate() == nil {
		t.Fatal("accepted expression injection")
	}
	if (Filters{}).Validate() == nil {
		t.Fatal("accepted no media")
	}
	f = Filters{Images: true, Extensions: []string{"jpg", "jpeg", "png"}}
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestWindowsSegment(t *testing.T) {
	for in, want := range map[string]string{"CON": "_CON", "a/b:c": "a_b_c", "../": "_", "LPT1.txt": "_LPT1.txt", "art. ": "art"} {
		if got := SafeSegment(in); got != want {
			t.Errorf("%q => %q want %q", in, got, want)
		}
	}
}
