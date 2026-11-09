package oauthflow

import (
	"context"
	"dl-manager/internal/downloader"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOAuthChild(t *testing.T) {
	if os.Getenv("DLM_OAUTH_CHILD") != "1" {
		return
	}
	mode := os.Getenv("DLM_OAUTH_MODE")
	if mode == "wait" {
		fmt.Println("https://www.tumblr.com/oauth/authorize?oauth_token=public-request")
		time.Sleep(30 * time.Second)
		return
	}
	if mode == "fail" {
		fmt.Fprintln(os.Stderr, "secret-in-error-DO-NOT-EXPOSE")
		os.Exit(4)
	}
	if mode == "large" {
		fmt.Print(strings.Repeat("x", 70000))
		return
	}
	fmt.Print("https://www.deviantart.com/oauth2/authorize?state=test&redirect_uri=https%3A%2F%2Fmikf.github.io%2Fgallery-dl%2Foauth-redirect.html\n")
	fmt.Print("Your 'refresh-")
	fmt.Print("token' is\n\nSYNTHETIC_REFRESH\n\n")
}
func TestParserRequiresExpectedSiteAndCompleteCredentials(t *testing.T) {
	p := newOutput("deviantart")
	p.Write([]byte("https://evil.example/oauth2/authorize\nhttps://www.tumblr.com/oauth/authorize\n"))
	if p.url != "" {
		t.Fatal("foreign authorization URL accepted")
	}
	p.Write([]byte("https://www.deviantart.com/oauth2/authorize?state=test&redirect_uri=https%3A%2F%2Fmikf.github.io%2Fgallery-dl%2Foauth-redirect.html\nYour 'refresh-"))
	p.Write([]byte("token' is\r\n\r\nSYNTHETIC_REFRESH\r\n\r\n"))
	c, e := p.credentials()
	if e != nil || c.Method != "token" || c.Token != "SYNTHETIC_REFRESH" || c.TokenSecret != "" {
		t.Fatalf("unexpected credentials: %v", e)
	}
	q := newOutput("tumblr")
	q.Write([]byte("Your 'access-token' and 'access-token-secret' are\n\nFIRST\nSECOND\n\n"))
	c, e = q.credentials()
	if e != nil || c.Token != "FIRST" || c.TokenSecret != "SECOND" || c.Method != "oauth1" {
		t.Fatal("missing pair")
	}
	q = newOutput("tumblr")
	q.Write([]byte("Your 'access-token' and 'access-token-secret' are\n\nFIRST\n"))
	if _, e = q.credentials(); e == nil {
		t.Fatal("partial credentials accepted")
	}
}
func TestSessionResultIsOneUseAndNeverInStatus(t *testing.T) {
	t.Setenv("DLM_OAUTH_CHILD", "1")
	t.Setenv("DLM_OAUTH_MODE", "ok")
	s := New(t.TempDir())
	defer s.Close()
	id, e := s.Start("deviantart", downloader.Engine{Path: os.Args[0], Prefix: []string{"-test.run=TestOAuthChild", "--"}})
	if e != nil {
		t.Fatal(e)
	}
	waitState(t, s, id, "completed")
	state, e := s.Status(id)
	if e != nil || strings.Contains(fmt.Sprint(state), "SYNTHETIC_REFRESH") {
		t.Fatal("secret leaked through status")
	}
	c, e := s.Result(id)
	if e != nil || c.Token != "SYNTHETIC_REFRESH" {
		t.Fatal("missing result", e)
	}
	if _, e = s.Result(id); e == nil {
		t.Fatal("result reused")
	}
	entries, _ := os.ReadDir(s.root)
	if len(entries) != 0 {
		t.Fatal("temporary authorization files retained")
	}
}
func TestCancelAndCloseStopProcessAndReleaseSession(t *testing.T) {
	t.Setenv("DLM_OAUTH_CHILD", "1")
	t.Setenv("DLM_OAUTH_MODE", "wait")
	s := New(t.TempDir())
	defer s.Close()
	eng := downloader.Engine{Path: os.Args[0], Prefix: []string{"-test.run=TestOAuthChild", "--"}}
	id, e := s.Start("tumblr", eng)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Start("deviantart", eng); e == nil {
		t.Fatal("concurrent authorization accepted")
	}
	if e = s.Cancel(id); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Result(id); e == nil {
		t.Fatal("cancelled result exposed")
	}
	id, e = s.Start("tumblr", eng)
	if e != nil {
		t.Fatal("session not released", e)
	}
	s.Close()
	if _, e = s.Status(id); e == nil {
		t.Fatal("closed session retained")
	}
	if _, e = s.Start("tumblr", eng); e == nil {
		t.Fatal("closed service restarted")
	}
}
func TestFailureTimeoutAndOversizedOutputDoNotLeak(t *testing.T) {
	for _, mode := range []string{"fail", "large", "wait"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("DLM_OAUTH_CHILD", "1")
			t.Setenv("DLM_OAUTH_MODE", mode)
			s := New(t.TempDir())
			if mode == "wait" {
				s.timeout = 100 * time.Millisecond
			}
			defer s.Close()
			id, e := s.Start("deviantart", downloader.Engine{Path: os.Args[0], Prefix: []string{"-test.run=TestOAuthChild", "--"}})
			if e != nil {
				t.Fatal(e)
			}
			waitState(t, s, id, "failed")
			v, e := s.Status(id)
			if e != nil || v.Error == "" || strings.Contains(v.Error, "secret-in-error") {
				t.Fatal("unsafe failure", e)
			}
			if _, e = s.Result(id); e == nil {
				t.Fatal("failed result")
			}
		})
	}
}
func TestUnsupportedAndUnsafeAuthorizationURLs(t *testing.T) {
	s := New(t.TempDir())
	defer s.Close()
	if _, e := s.Start("pixiv", downloader.Engine{}); e == nil {
		t.Fatal("unsupported site")
	}
	for _, u := range []string{"http://www.tumblr.com/oauth/authorize", "https://www.tumblr.com.evil.test/oauth/authorize", "https://user@www.tumblr.com/oauth/authorize", "https://www.tumblr.com:444/oauth/authorize", "https://www.tumblr.com/logout"} {
		if ValidURL("tumblr", u) {
			t.Fatal("unsafe URL", u)
		}
	}
}
func waitState(t *testing.T, s *Service, id, want string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		v, e := s.Status(id)
		if e != nil {
			t.Fatal(e)
		}
		if v.State == want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("state %s, wanted %s", v.State, want)
		case <-time.After(10 * time.Millisecond):
		}
	}
}
func TestCompletedCredentialsExpireAndWrongCancelDoesNotStopSession(t *testing.T) {
	t.Setenv("DLM_OAUTH_CHILD", "1")
	t.Setenv("DLM_OAUTH_MODE", "ok")
	s := New(t.TempDir())
	s.timeout = 1500 * time.Millisecond
	defer s.Close()
	id, e := s.Start("deviantart", downloader.Engine{Path: os.Args[0], Prefix: []string{"-test.run=TestOAuthChild", "--"}})
	if e != nil {
		t.Fatal(e)
	}
	s.Cancel("wrong")
	waitState(t, s, id, "completed")
	waitState(t, s, id, "failed")
	if _, e = s.Result(id); e == nil {
		t.Fatal("expired token exposed")
	}
	if _, e = s.Start("deviantart", downloader.Engine{Path: os.Args[0], Prefix: []string{"-test.run=TestOAuthChild", "--"}}); e != nil {
		t.Fatal("expired session blocked a new authorization", e)
	}
}
