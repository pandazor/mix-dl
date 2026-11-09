package downloader

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Engine struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Prefix  []string `json:"-"`
	Version string   `json:"version"`
	Error   string   `json:"error,omitempty"`
}

func probe(name, path string, prefix []string) (Engine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	args := append(append([]string{}, prefix...), "--version")
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, e := cmd.CombinedOutput()
	version := strings.TrimSpace(string(out))
	if e != nil || version == "" {
		return Engine{}, errors.New("Moteur inutilisable : " + path)
	}
	return Engine{Name: name, Path: path, Prefix: prefix, Version: strings.Split(version, "\n")[0]}, nil
}
func Resolve(name, override, toolsDir string) (Engine, error) {
	if override != "" {
		return probe(name, override, nil)
	}
	managed := filepath.Join(toolsDir, name+".exe")
	if _, e := os.Stat(managed); e == nil {
		return probe(name, managed, nil)
	}
	if path, e := exec.LookPath(name); e == nil {
		if v, e := probe(name, path, nil); e == nil {
			return v, nil
		}
	}
	module := "gallery_dl"
	if name == "yt-dlp" {
		module = "yt_dlp"
	}
	for _, interpreter := range []string{"py", "python"} {
		if p, e := exec.LookPath(interpreter); e == nil {
			if v, e := probe(name, p, []string{"-m", module}); e == nil {
				return v, nil
			}
		}
	}
	return Engine{}, errors.New(name + " absent : utilisez le paquet complet ou indiquez son exécutable dans les paramètres")
}
