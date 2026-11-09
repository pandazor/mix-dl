package web

import (
	"dl-manager/internal/downloader"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOAuthAPIChild(t *testing.T) {
	if os.Getenv("DLM_WEB_OAUTH") != "1" {
		return
	}
	fmt.Print("Your 'refresh-token' is\n\nSYNTHETIC_WEB_TOKEN\n\n")
}
func TestOAuthRoutesAreProtectedAndConsumeResult(t *testing.T) {
	a, _ := profileAPI(t)
	defer a.Close()
	for _, action := range []string{"start", "status", "result", "cancel", "open"} {
		r := httptest.NewRequest("POST", "/api/oauth/"+action, strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("unprotected route", action, w.Code)
		}
	}
	if w := request(t, a, "POST", "oauth/start", `{"site":"pixiv"}`); w.Code == 200 {
		t.Fatal("unsupported OAuth")
	}
	t.Setenv("DLM_WEB_OAUTH", "1")
	id, e := a.OAuth.Start("deviantart", downloader.Engine{Path: os.Args[0], Prefix: []string{"-test.run=TestOAuthAPIChild", "--"}})
	if e != nil {
		t.Fatal(e)
	}
	body := `{"id":"` + id + `"}`
	until := time.Now().Add(5 * time.Second)
	for {
		w := request(t, a, "POST", "oauth/status", body)
		if w.Code != 200 || strings.Contains(w.Body.String(), "SYNTHETIC_WEB_TOKEN") {
			t.Fatal("status leaked or failed", w.Code)
		}
		if strings.Contains(w.Body.String(), `"state":"completed"`) {
			break
		}
		if time.Now().After(until) {
			t.Fatal("timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if w := request(t, a, "POST", "oauth/result", `{"id":"wrong"}`); w.Code == 200 {
		t.Fatal("wrong session consumed token")
	}
	w := request(t, a, "POST", "oauth/result", body)
	var c downloader.Credentials
	if json.Unmarshal(w.Body.Bytes(), &c) != nil || c.Token != "SYNTHETIC_WEB_TOKEN" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing safe result")
	}
	if w = request(t, a, "POST", "oauth/result", body); w.Code == 200 {
		t.Fatal("result reused")
	}
	if w = request(t, a, "GET", "state", ""); strings.Contains(w.Body.String(), "SYNTHETIC_WEB_TOKEN") {
		t.Fatal("state leaked secret")
	}
	state, _ := a.Store.Snapshot()
	if len(state.Profiles) != 0 {
		t.Fatal("OAuth unexpectedly saved account")
	}
}
