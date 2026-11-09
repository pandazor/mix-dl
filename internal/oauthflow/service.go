package oauthflow

import (
	"bytes"
	"context"
	"dl-manager/internal/auth"
	"dl-manager/internal/downloader"
	"dl-manager/internal/model"
	"dl-manager/internal/process"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const maxOutput = 64 * 1024

type Status struct {
	ID    string `json:"id"`
	State string `json:"state"`
	URL   string `json:"url,omitempty"`
	Error string `json:"error,omitempty"`
}
type session struct {
	id, site, state, message string
	output                   *output
	credentials              downloader.Credentials
	cancel                   context.CancelFunc
	done                     chan struct{}
	expires                  time.Time
}
type Service struct {
	mu      sync.Mutex
	root    string
	timeout time.Duration
	current *session
	closed  bool
}

func New(root string) *Service { return &Service{root: root, timeout: 5 * time.Minute} }
func ValidURL(site, raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || len(raw) > 8192 || u.Scheme != "https" || u.User != nil || u.Fragment != "" {
		return false
	}
	if site == "tumblr" {
		return u.Host == "www.tumblr.com" && u.Path == "/oauth/authorize" && u.Query().Get("oauth_token") != ""
	}
	return site == "deviantart" && u.Host == "www.deviantart.com" && u.Path == "/oauth2/authorize" && u.Query().Get("redirect_uri") == "https://mikf.github.io/gallery-dl/oauth-redirect.html" && u.Query().Get("state") != ""
}
func (s *Service) Start(site string, engine downloader.Engine) (string, error) {
	if site != "tumblr" && site != "deviantart" {
		return "", errors.New("Connexion guidée disponible pour Tumblr et DeviantArt")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", errors.New("Application en cours de fermeture")
	}
	if q := s.current; q != nil && (q.state == "failed" || (q.state == "completed" && time.Now().After(q.expires))) {
		q.cancel()
		q.credentials = downloader.Credentials{}
		s.current = nil
	}
	if s.current != nil {
		return "", errors.New("Terminez ou annulez la connexion en cours")
	}
	if err := auth.PrivateDir(s.root); err != nil {
		return "", errors.New("Dossier temporaire OAuth inaccessible")
	}
	dir, err := os.MkdirTemp(s.root, "oauth-")
	if err != nil {
		return "", errors.New("Dossier temporaire OAuth inaccessible")
	}
	if err = auth.PrivateDir(dir); err != nil {
		os.RemoveAll(dir)
		return "", errors.New("Protection du dossier OAuth impossible")
	}
	config := map[string]any{"cache": map[string]any{"file": nil}, "extractor": map[string]any{"oauth": map[string]any{"cache": false, "browser": false, "host": "127.0.0.1"}}, "output": map[string]any{"colors": false, "stdout": map[string]any{"encoding": "utf-8", "line_buffering": true}, "stderr": map[string]any{"encoding": "utf-8", "line_buffering": true}}}
	raw, _ := json.Marshal(config)
	path := filepath.Join(dir, "config.json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		os.RemoveAll(dir)
		return "", errors.New("Configuration OAuth impossible")
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	q := &session{id: model.ID(), site: site, state: "starting", output: newOutput(site), cancel: cancel, done: make(chan struct{}), expires: time.Now().Add(s.timeout)}
	args := append(append([]string{}, engine.Prefix...), "--config-ignore", "--config", path, "oauth:"+site)
	cmd := exec.Command(engine.Path, args...)
	cmd.Dir = dir
	cmd.Stdout = q.output
	cmd.Stderr = q.output
	tree, err := process.Start(cmd)
	if err != nil {
		cancel()
		os.RemoveAll(dir)
		return "", errors.New("Impossible de démarrer gallery-dl pour la connexion")
	}
	s.current = q
	go func() {
		err := tree.Wait(ctx)
		creds, parseErr := q.output.credentials()
		q.output.wipe()
		os.RemoveAll(dir)
		s.mu.Lock()
		if s.current == q && q.state != "cancelled" {
			if err != nil || parseErr != nil {
				q.state = "failed"
				q.message = "Connexion non terminée : autorisation refusée, port 6414 occupé ou réponse du site invalide. Réessayez."
				if errors.Is(err, context.DeadlineExceeded) {
					q.message = "Connexion expirée après cinq minutes. Réessayez."
				}
			} else {
				q.state = "completed"
				q.credentials = creds
			}
		}
		s.mu.Unlock()
		close(q.done)
	}()
	go func() {
		<-ctx.Done()
		<-q.done
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.current == q && time.Now().After(q.expires) {
			q.credentials = downloader.Credentials{}
			q.state = "failed"
			q.message = "Connexion expirée. Relancez la génération."
			q.output.wipe()
		}
	}()
	return q.id, nil
}
func (s *Service) Status(id string) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := s.current
	if q == nil || q.id != id {
		return Status{}, errors.New("Connexion introuvable ou terminée")
	}
	v := Status{ID: q.id, State: q.state, Error: q.message}
	if q.state == "starting" {
		v.URL = q.output.authorizationURL()
		if v.URL != "" {
			v.State = "waiting"
		}
	}
	return v, nil
}
func (s *Service) Result(id string) (downloader.Credentials, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := s.current
	if q == nil || q.id != id || q.state != "completed" || time.Now().After(q.expires) {
		return downloader.Credentials{}, errors.New("Aucun token disponible pour cette connexion")
	}
	c := q.credentials
	q.credentials = downloader.Credentials{}
	q.cancel()
	s.current = nil
	return c, nil
}
func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	q := s.current
	if q == nil || q.id != id {
		s.mu.Unlock()
		return nil
	}
	q.state = "cancelled"
	q.cancel()
	q.credentials = downloader.Credentials{}
	s.mu.Unlock()
	<-q.done
	s.mu.Lock()
	if s.current == q {
		q.output.wipe()
		q.credentials = downloader.Credentials{}
		s.current = nil
	}
	s.mu.Unlock()
	return nil
}
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	q := s.current
	s.mu.Unlock()
	if q != nil {
		s.Cancel(q.id)
	}
}

type output struct {
	mu        sync.Mutex
	site, url string
	buf       []byte
	overflow  bool
}

func newOutput(site string) *output { return &output{site: site} }
func (p *output) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.buf)+len(b) > maxOutput {
		p.overflow = true
		return len(b), nil
	}
	p.buf = append(p.buf, b...)
	if p.url == "" {
		for _, line := range bytes.Split(p.buf, []byte("\n")) {
			raw := strings.TrimSpace(string(line))
			if ValidURL(p.site, raw) {
				p.url = raw
				break
			}
		}
	}
	return len(b), nil
}
func (p *output) authorizationURL() string { p.mu.Lock(); defer p.mu.Unlock(); return p.url }

var single = regexp.MustCompile(`(?m)^Your 'refresh-token' is\n\n([^\n]+)\n\n`)
var pair = regexp.MustCompile(`(?m)^Your 'access-token' and 'access-token-secret' are\n\n([^\n]+)\n([^\n]+)\n\n`)
var tokenValue = regexp.MustCompile(`^[a-zA-Z0-9._~+/=-]+$`)

func (p *output) credentials() (downloader.Credentials, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	bad := errors.New("Réponse OAuth invalide")
	if p.overflow {
		return downloader.Credentials{}, bad
	}
	raw := strings.ReplaceAll(string(p.buf), "\r\n", "\n")
	if p.site == "deviantart" {
		m := single.FindStringSubmatch(raw)
		if len(m) == 2 && len(m[1]) <= 16384 && tokenValue.MatchString(m[1]) {
			return downloader.Credentials{Method: "token", Token: m[1]}, nil
		}
	}
	if p.site == "tumblr" {
		m := pair.FindStringSubmatch(raw)
		if len(m) == 3 && len(m[1]) <= 16384 && len(m[2]) <= 16384 && tokenValue.MatchString(m[1]) && tokenValue.MatchString(m[2]) {
			return downloader.Credentials{Method: "oauth1", Token: m[1], TokenSecret: m[2]}, nil
		}
	}
	return downloader.Credentials{}, bad
}
func (p *output) wipe() { p.mu.Lock(); defer p.mu.Unlock(); clear(p.buf); p.buf = nil; p.url = "" }
