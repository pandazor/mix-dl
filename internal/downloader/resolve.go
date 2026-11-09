package downloader

import (
	"context"
	"dl-manager/internal/process"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Engine struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Prefix  []string `json:"-"`
	Version string   `json:"version"`
	Error   string   `json:"error,omitempty"`
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func probe(name, path string, prefix []string) (Engine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	args := append(append([]string{}, prefix...), "--version")
	out, e := process.ProbeOutput(ctx, path, args...)
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
	managed := filepath.Join(toolsDir, executableName(name))
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
	interpreters := []string{"python3", "python"}
	if runtime.GOOS == "windows" {
		interpreters = []string{"py", "python"}
	}
	for _, interpreter := range interpreters {
		if p, e := exec.LookPath(interpreter); e == nil {
			if v, e := probe(name, p, []string{"-m", module}); e == nil {
				return v, nil
			}
		}
	}
	return Engine{}, errors.New(name + " absent : utilisez le paquet complet ou indiquez son exécutable dans les paramètres")
}
