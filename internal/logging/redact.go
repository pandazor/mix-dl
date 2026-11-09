package logging

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Redactor struct{ secrets []string }

func NewRedactor(values []string) *Redactor {
	r := &Redactor{}
	for _, v := range values {
		if v != "" {
			r.secrets = append(r.secrets, v, url.QueryEscape(v), url.PathEscape(v))
		}
	}
	return r
}

var urls = regexp.MustCompile(`https?://[^\s<>"']+`)
var headers = regexp.MustCompile(`(?i)(authorization|cookie|set-cookie|api[-_]?key|access[-_]?token|refresh[-_]?token|password)\s*[:=]\s*[^\r\n]+`)

func (r *Redactor) Redact(s string) string {
	for _, v := range r.secrets {
		s = strings.ReplaceAll(s, v, "[masqué]")
	}
	s = headers.ReplaceAllString(s, "$1: [masqué]")
	s = urls.ReplaceAllStringFunc(s, func(raw string) string {
		u, e := url.Parse(raw)
		if e != nil {
			return "[URL]"
		}
		u.User = nil
		if u.RawQuery != "" {
			u.RawQuery = "parametres-masques"
		}
		u.Fragment = ""
		return u.String()
	})
	return s
}

type Rotator struct {
	mu  sync.Mutex
	Dir string
}

func (r *Rotator) Write(line string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e := os.MkdirAll(r.Dir, 0700); e != nil {
		return e
	}
	p := filepath.Join(r.Dir, "application.log")
	if info, e := os.Stat(p); e == nil && info.Size() > 5*1024*1024 {
		os.Remove(p + ".3")
		for i := 2; i >= 1; i-- {
			os.Rename(fmt.Sprintf("%s.%d", p, i), fmt.Sprintf("%s.%d", p, i+1))
		}
		if e = os.Rename(p, p+".1"); e != nil {
			return e
		}
	}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	_, e = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
	ce := f.Close()
	if e != nil {
		return e
	}
	return ce
}
