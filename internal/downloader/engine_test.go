package downloader

import (
	"dl-manager/internal/model"
	"encoding/json"
	"strings"
	"testing"
)

func TestInvocationSecretsNotInArguments(t *testing.T) {
	j := model.Job{ID: "test", URL: "https://rule34.xxx/index.php?page=post&s=list&tags=landscape", Site: "rule34", Destination: t.TempDir(), Filters: model.DefaultFilters(), Layout: "creator"}
	c := Credentials{Method: "api", UserID: "123", Token: "test-secret"}
	inv, e := BuildInvocation(j, model.DefaultSettings(), c, t.TempDir(), "archive.db", t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	args := strings.Join(inv.Args, " ")
	if strings.Contains(args, c.Token) || strings.Contains(args, c.UserID) {
		t.Fatal("secret in process arguments")
	}
	var cfg map[string]any
	if e = json.Unmarshal(inv.Config, &cfg); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(inv.Config), c.Token) {
		t.Fatal("missing configured auth")
	}
	if inv.Args[len(inv.Args)-1] != j.URL || inv.Args[len(inv.Args)-2] != "--" {
		t.Fatal("missing URL option delimiter")
	}
}
func Test403IsNotAutomaticallyAuthentication(t *testing.T) {
	if Classify("HTTP Error 403: Forbidden").Status() == "authentication_required" {
		t.Fatal("403 misclassified")
	}
	if Classify("[instagram][error] AuthRequired: login required").Status() != "authentication_required" {
		t.Fatal("auth error not recognized")
	}
	if Classify("HTTP Error 429: Too Many Requests").Kind != "rate_limit" {
		t.Fatal("429 not classified")
	}
}
func TestYouTubeHasArchiveAndNoShell(t *testing.T) {
	j := model.Job{ID: "yt", Site: "youtube", URL: "https://youtu.be/abcdef", Destination: t.TempDir(), Filters: model.DefaultFilters(), Layout: "creator"}
	v, e := BuildInvocation(j, model.DefaultSettings(), Credentials{}, t.TempDir(), "archive.txt", t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	a := strings.Join(v.Args, " ")
	if !strings.Contains(a, "--download-archive") || !strings.Contains(a, "--continue") || strings.Contains(a, "--exec") {
		t.Fatal(a)
	}
}
