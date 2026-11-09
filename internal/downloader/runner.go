package downloader

import (
	"bufio"
	"context"
	"crypto/sha256"
	"dl-manager/internal/auth"
	"dl-manager/internal/logging"
	"dl-manager/internal/model"
	"dl-manager/internal/process"
	"dl-manager/internal/storage"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	Store    *storage.Store
	Vault    *auth.Vault
	DataDir  string
	ToolsDir string
	Logs     *logging.Rotator
}

func (r *Runner) Run(ctx context.Context, j model.Job, emit func(model.Event)) (result error) {
	state, e := r.Store.Snapshot()
	if e != nil {
		return e
	}
	s := state.Settings
	name, override := "gallery-dl", s.GalleryPath
	if j.Site == "youtube" {
		name, override = "yt-dlp", s.YTDLPPath
	}
	engine, e := Resolve(name, override, r.ToolsDir)
	if e != nil {
		return e
	}
	var creds Credentials
	if j.ProfileID != "" {
		payload, e := r.Vault.Load(j.ProfileID)
		if e != nil {
			return &DownloadError{"authentication", "Impossible de lire le compte : reconfigurez-le dans Comptes"}
		}
		if e = json.Unmarshal(payload, &creds); e != nil {
			return e
		}
	}
	secrets := []string{creds.Token, creds.TokenSecret}
	for _, l := range strings.Split(creds.Cookies, "\n") {
		p := strings.Split(strings.TrimSpace(l), "\t")
		if len(p) == 7 {
			secrets = append(secrets, p[6])
		}
	}
	redactor := logging.NewRedactor(secrets)
	log := func(line string) {
		line = redactor.Redact(line)
		if len(line) > 8192 {
			line = line[:8192]
		}
		emit(model.Event{Kind: "log", Message: line})
		if r.Logs != nil {
			if err := r.Logs.Write(j.ID + " " + line); err != nil {
				emit(model.Event{Kind: "log", Message: "Écriture du journal impossible"})
			}
		}
	}
	if e = os.MkdirAll(j.Destination, 0755); e != nil {
		return fmt.Errorf("Destination inaccessible : %w", e)
	}
	test, e := os.CreateTemp(j.Destination, ".dl-manager-write-test-")
	if e != nil {
		return fmt.Errorf("Destination non inscriptible : %w", e)
	}
	test.Close()
	os.Remove(test.Name())
	tempRoot := filepath.Join(r.DataDir, "temporary")
	if e = auth.PrivateDir(tempRoot); e != nil {
		return e
	}
	temp, e := os.MkdirTemp(tempRoot, "run-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(temp)
	cacheKey := j.ProfileID + "-cache"
	if j.ProfileID != "" {
		if cache, err := r.Vault.Load(cacheKey); err == nil {
			if e = os.WriteFile(filepath.Join(temp, "cache.sqlite3"), cache, 0600); e != nil {
				return e
			}
		}
	}
	defer func() {
		if j.ProfileID != "" {
			if cache, err := os.ReadFile(filepath.Join(temp, "cache.sqlite3")); err == nil {
				if err = r.Vault.Save(cacheKey, cache); err != nil && result == nil {
					result = errors.New("Le cache de connexion n'a pas pu être sauvegardé de façon chiffrée")
				}
			}
		}
	}()
	hash := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(j.Destination)) + "|" + j.Site))
	archiveDir := filepath.Join(r.DataDir, "archives")
	if e = os.MkdirAll(archiveDir, 0700); e != nil {
		return e
	}
	suffix := ".sqlite3"
	if name == "yt-dlp" {
		suffix = ".txt"
	}
	archive := filepath.Join(archiveDir, name+"-"+hex.EncodeToString(hash[:8])+suffix)
	inv, e := BuildInvocation(j, s, creds, temp, archive, r.ToolsDir)
	if e != nil {
		return e
	}
	if len(inv.Config) > 0 {
		if e = os.WriteFile(filepath.Join(temp, "config.json"), inv.Config, 0600); e != nil {
			return e
		}
	}
	if len(inv.Cookies) > 0 {
		if e = os.WriteFile(filepath.Join(temp, "cookies.txt"), inv.Cookies, 0600); e != nil {
			return e
		}
	}
	args := append(append([]string{}, engine.Prefix...), inv.Args...)
	cmd := exec.Command(engine.Path, args...)
	cmd.Dir = r.ToolsDir
	if _, e = os.Stat(cmd.Dir); e != nil {
		cmd.Dir = temp
	}
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8", "PYTHONUNBUFFERED=1", "NO_COLOR=1", "PATH="+r.ToolsDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	baselines := partialBaselines(j.Destination)
	reader, writer, e := os.Pipe()
	if e != nil {
		return e
	}
	defer reader.Close()
	cmd.Stdout = writer
	cmd.Stderr = writer
	tree, e := process.Start(cmd)
	writer.Close()
	if e != nil {
		return e
	}
	tracker := &fileTracker{emit: emit, baselines: baselines}
	scanDone := make(chan error, 1)
	var lastError string
	hadError := false
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 4096), 2*1024*1024)
		scanner.Split(splitLines)
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "DLM_START:"):
				tracker.begin(strings.TrimPrefix(line, "DLM_START:"))
			case strings.HasPrefix(line, "DLM_SUCCESS:"):
				tracker.finish(strings.TrimPrefix(line, "DLM_SUCCESS:"))
				emit(model.Event{Kind: "file"})
			case strings.HasPrefix(line, "DLM_SKIP:"):
				emit(model.Event{Kind: "skip"})
			case strings.HasPrefix(line, "DLM_BEGIN:"):
				var p string
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "DLM_BEGIN:")), &p) == nil {
					tracker.begin(p)
				}
			case strings.HasPrefix(line, "DLM_YT:"):
				bits := strings.SplitN(strings.TrimPrefix(line, "DLM_YT:"), "\t", 2)
				if len(bits) == 2 {
					var p string
					n, err := strconv.ParseInt(bits[1], 10, 64)
					if err == nil && json.Unmarshal([]byte(bits[0]), &p) == nil {
						tracker.progress(p, n)
					}
				}
			case strings.HasPrefix(line, "DLM_END:"):
				var p string
				if json.Unmarshal([]byte(strings.TrimPrefix(line, "DLM_END:")), &p) == nil {
					tracker.finish(p)
					emit(model.Event{Kind: "file"})
				}
			default:
				line = strings.TrimSpace(line)
				if line != "" {
					lower := strings.ToLower(line)
					if strings.Contains(lower, "[error]") || strings.HasPrefix(lower, "error:") {
						hadError = true
						lastError = redactor.Redact(line)
					}
					log(line)
				}
			}
		}
		scanDone <- scanner.Err()
	}()
	pollCtx, cancelPoll := context.WithCancel(context.Background())
	var pollWG sync.WaitGroup
	pollWG.Add(1)
	go func() {
		defer pollWG.Done()
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-pollCtx.Done():
				return
			case <-ticker.C:
				tracker.poll()
			}
		}
	}()
	log("Moteur : " + engine.Name + " " + engine.Version)
	waitErr := tree.Wait(ctx)
	scanErr := <-scanDone
	cancelPoll()
	pollWG.Wait()
	tracker.poll()
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	emit(model.Event{Kind: "exit", ExitCode: code})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if scanErr != nil {
		return fmt.Errorf("Lecture du moteur : %w", scanErr)
	}
	if waitErr != nil || hadError {
		if lastError == "" {
			lastError = fmt.Sprintf("%s s'est arrêté avec le code %d. Consultez le journal.", name, code)
		}
		return Classify(lastError)
	}
	return nil
}
func splitLines(data []byte, atEOF bool) (int, []byte, error) {
	if i := strings.IndexAny(string(data), "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

type fileTracker struct {
	mu        sync.Mutex
	path      string
	base      int64
	seen      int64
	emit      func(model.Event)
	yt        map[string]int64
	baselines map[string]int64
}

func size(p string) int64 {
	if v, e := os.Stat(p); e == nil && !v.IsDir() {
		return v.Size()
	}
	return 0
}
func (t *fileTracker) begin(p string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pollLocked()
	t.path = p
	t.base = size(p + ".part")
	if t.baselines != nil {
		t.base = t.baselines[p]
	}
	t.seen = t.base
	t.yt = map[string]int64{}
	t.emit(model.Event{Kind: "current", Message: filepath.Base(p)})
}
func (t *fileTracker) observe(n int64) {
	if n > t.seen {
		t.emit(model.Event{Kind: "bytes", Bytes: n - t.seen})
		t.seen = n
	}
}
func (t *fileTracker) pollLocked() {
	if t.path == "" {
		return
	}
	n := size(t.path + ".part")
	if n == 0 {
		n = size(t.path)
	}
	t.observe(n)
}
func (t *fileTracker) poll() { t.mu.Lock(); defer t.mu.Unlock(); t.pollLocked() }
func (t *fileTracker) finish(p string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.yt) == 0 {
		t.observe(size(p))
	}
	t.path = ""
}
func (t *fileTracker) progress(p string, n int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.yt == nil {
		t.yt = map[string]int64{}
	}
	old, ok := t.yt[p]
	if !ok {
		old = t.baselines[p]
		if p == t.path && t.seen > old {
			old = t.seen
		}
	}
	if n > old {
		t.emit(model.Event{Kind: "bytes", Bytes: n - old})
		t.yt[p] = n
	}
	t.path = ""
}
func partialBaselines(root string) map[string]int64 {
	out := map[string]int64{}
	filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(p, ".part") {
			out[strings.TrimSuffix(p, ".part")] = size(p)
		}
		return nil
	})
	return out
}

var _ io.Reader = (*os.File)(nil)
