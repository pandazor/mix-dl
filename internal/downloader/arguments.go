package downloader

import (
	"dl-manager/internal/model"
	"dl-manager/internal/platform"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Credentials struct {
	Method      string `json:"method"`
	UserID      string `json:"userId"`
	Token       string `json:"token"`
	TokenSecret string `json:"tokenSecret"`
	Cookies     string `json:"cookies"`
}
type Invocation struct {
	Args    []string
	Config  []byte
	Cookies []byte
}

func BuildInvocation(j model.Job, s model.Settings, c Credentials, temp, archive, toolsDir string) (Invocation, error) {
	if _, e := model.ParseURL(j.URL); e != nil {
		return Invocation{}, e
	}
	if e := j.Filters.Validate(); e != nil {
		return Invocation{}, e
	}
	allowed := j.Filters.Allowed()
	if len(allowed) == 0 {
		return Invocation{}, errors.New("Aucune extension sélectionnée")
	}
	inv := Invocation{}
	cookiePath := filepath.Join(temp, "cookies.txt")
	cfgPath := filepath.Join(temp, "config.json")
	if j.Site == "youtube" {
		if !j.Filters.Videos {
			return inv, errors.New("YouTube : activez Vidéos pour télécharger une vidéo")
		}
		path := j.Destination
		switch j.Layout {
		case "site":
			path = filepath.Join(path, "youtube")
		case "creator":
			path = filepath.Join(path, "youtube", "%(uploader_id,uploader,channel_id|inconnu)s")
		}
		output := filepath.Join(path, "%(id)s_%(title).120B.%(ext)s")
		inv.Args = []string{"--ignore-config", "--no-colors", "--newline", "--no-playlist", "--yes-playlist", "--continue", "--no-overwrites", "--windows-filenames", "--download-archive", archive, "--output", output, "--progress", "--progress-template", "download:DLM_YT:%(progress.filename)j\t%(progress.downloaded_bytes)s", "--print", "before_dl:DLM_BEGIN:%(filename)j", "--print", "after_move:DLM_END:%(filepath)j", "--no-simulate", "--socket-timeout", "30", "--retries", "3", "--fragment-retries", "3"}
		if ffmpeg, err := os.Stat(filepath.Join(toolsDir, executableName("ffmpeg"))); err == nil && ffmpeg.Mode().IsRegular() {
			inv.Args = append(inv.Args, "--ffmpeg-location", toolsDir)
		}
		deno := filepath.Join(toolsDir, executableName("deno"))
		if info, err := os.Stat(deno); err == nil && info.Mode().IsRegular() {
			inv.Args = append(inv.Args, "--js-runtimes", "deno:"+deno)
		}
		formats := []string{}
		for _, e := range allowed {
			if e == "mp4" {
				formats = append(formats, "bv[ext=mp4]+ba[ext=m4a]/b[ext=mp4]")
			} else if e == "webm" {
				formats = append(formats, "bv[ext=webm]+ba[ext=webm]/b[ext=webm]")
			} else {
				formats = append(formats, "b[ext="+e+"]")
			}
		}
		inv.Args = append(inv.Args, "--format", strings.Join(formats, "/"))
		if c.Method == "cookies" {
			inv.Args = append(inv.Args, "--cookies", cookiePath)
			inv.Cookies = []byte(c.Cookies)
		}
		if s.RateKiB > 0 {
			inv.Args = append(inv.Args, "--limit-rate", strconv.FormatInt(s.RateKiB, 10)+"K")
		}
		inv.Args = append(inv.Args, "--", j.URL)
		return inv, nil
	}
	u, _ := url.Parse(j.URL)
	creator := platform.Get(j.Site).Creator
	if j.Site == "rule34" {
		tag := u.Query().Get("tags")
		if tag != "" {
			creator = strings.ReplaceAll(strings.ReplaceAll(model.SafeSegment(tag), "{", "{{"), "}", "}}")
		}
	}
	if j.Site == "generic" {
		creator = model.SafeSegment(u.Hostname())
	}
	dir := []string{}
	if j.Layout == "site" || j.Layout == "creator" {
		site := j.Site
		if site == "generic" {
			site = model.SafeSegment(u.Hostname())
		}
		dir = append(dir, site)
	}
	if j.Layout == "creator" {
		dir = append(dir, creator)
	}
	literals := []string{}
	for _, x := range allowed {
		literals = append(literals, strconv.Quote(x))
	}
	filter := "extension in (" + strings.Join(literals, ",") + ",)"
	extract := map[string]any{"base-directory": j.Destination, "directory": dir, "archive": archive, "skip": true, "path-restrict": "windows", "path-strip": "windows", "image-filter": filter, "retries": 3, "timeout": 30}
	if j.Filename == "id-title" {
		extract["filename"] = "{id|filename}_{num|0}_{title|filename}.{extension}"
		// Direct URLs have no post ID. Their full source path is their identity.
		extract["directlink"] = map[string]any{"filename": "{domain}/{path}/{filename}.{extension}"}
	}
	if c.Method == "cookies" {
		extract["cookies"] = cookiePath
		inv.Cookies = []byte(c.Cookies)
	}
	siteOpts := map[string]any{}
	switch c.Method {
	case "api":
		siteOpts["user-id"] = c.UserID
		siteOpts["api-key"] = c.Token
	case "token":
		siteOpts["refresh-token"] = c.Token
	case "oauth1":
		siteOpts["access-token"] = c.Token
		siteOpts["access-token-secret"] = c.TokenSecret
	}
	if j.Site != "generic" {
		extract[j.Site] = siteOpts
	}
	httpOpts := map[string]any{"part": true, "timeout": 30, "retries": 3}
	if s.RateKiB > 0 {
		httpOpts["rate"] = s.RateKiB * 1024
	}
	output := map[string]any{"shorten": false, "colors": false, "mode": map[string]string{"start": "DLM_START:{0}\n", "skip": "DLM_SKIP:{0}\n", "success": "DLM_SUCCESS:{0}\n", "progress": "", "progress-total": ""}, "stdout": map[string]any{"encoding": "utf-8", "line_buffering": true}, "stderr": map[string]any{"encoding": "utf-8", "line_buffering": true}}
	cfg := map[string]any{"extractor": extract, "downloader": map[string]any{"http": httpOpts, "ytdl": map[string]any{"module": "yt_dlp", "raw-options": map[string]any{"ffmpeg_location": toolsDir}}}, "output": output, "cache": map[string]any{"file": filepath.Join(temp, "cache.sqlite3")}}
	b, e := json.Marshal(cfg)
	if e != nil {
		return inv, e
	}
	inv.Config = b
	inv.Args = []string{"--config-ignore", "--config", cfgPath, "--no-input", "--", j.URL}
	return inv, nil
}

type DownloadError struct {
	Kind    string
	Message string
}

func (e *DownloadError) Error() string { return e.Message }
func (e *DownloadError) Status() string {
	if e.Kind == "authentication" {
		return "authentication_required"
	}
	return "failed"
}
func Classify(msg string) *DownloadError {
	lower := strings.ToLower(msg)
	kind := "download"
	switch {
	case strings.Contains(lower, "authrequired"), strings.Contains(lower, "authentication"), strings.Contains(lower, "login required"), strings.Contains(lower, "401"), strings.Contains(lower, "invalid token"), strings.Contains(lower, "expired session"):
		kind = "authentication"
	case strings.Contains(lower, "429"), strings.Contains(lower, "rate limit"):
		kind = "rate_limit"
	case strings.Contains(lower, "no space"), strings.Contains(lower, "disk full"), strings.Contains(lower, "errno 28"):
		kind = "disk_full"
	case strings.Contains(lower, "unsupported url"), strings.Contains(lower, "unsupportedurl"):
		kind = "unsupported"
	case strings.Contains(lower, "timed out"), strings.Contains(lower, "connection"), strings.Contains(lower, "network"):
		kind = "network"
	}
	return &DownloadError{kind, fmt.Sprintf("%s", msg)}
}
