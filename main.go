package main

import (
	"bytes"
	"dl-manager/internal/auth"
	"dl-manager/internal/downloader"
	"dl-manager/internal/logging"
	"dl-manager/internal/model"
	"dl-manager/internal/queue"
	"dl-manager/internal/storage"
	"dl-manager/internal/web"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"image"
	"image/color"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
)

//go:embed frontend/index.html frontend/style.css frontend/dist/app.js
var assets embed.FS

var errAlreadyOpen = errors.New("dl-manager is already running")

func main() {
	if e := run(); e != nil {
		log.Print(e)
		message(e.Error())
		os.Exit(1)
	}
}
func run() error {
	preview := flag.String("preview", "", "Developer UI preview: listen address (loopback only)")
	dataArg := flag.String("data", "", "Override portable data directory")
	var selfTest bool
	if runtime.GOOS == "windows" {
		flag.BoolVar(&selfTest, "self-test", false, "Test Windows encryption without credentials")
	}
	flag.Parse()
	executable, e := os.Executable()
	if e != nil {
		return e
	}
	root := filepath.Dir(executable)
	if *preview != "" {
		root, e = os.Getwd()
		if e != nil {
			return e
		}
	}
	dataDir := filepath.Join(root, "data")
	if *dataArg != "" {
		dataDir, e = filepath.Abs(*dataArg)
		if e != nil {
			return e
		}
	}
	if selfTest {
		tmp, e := os.MkdirTemp("", "dl-manager-self-test-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(tmp)
		v, e := auth.New(tmp)
		if e == nil {
			e = v.Save("probe", []byte("synthetic-test-only"))
		}
		if e == nil {
			b, err := v.Load("probe")
			e = err
			if e == nil && string(b) != "synthetic-test-only" {
				e = fmt.Errorf("DPAPI roundtrip mismatch")
			}
		}
		result := map[string]any{"dpapi": e == nil}
		if e != nil {
			result["error"] = e.Error()
		}
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		return os.WriteFile(filepath.Join(root, "self-test-result.json"), b, 0600)
	}
	if e = os.MkdirAll(dataDir, 0700); e != nil {
		return fmt.Errorf("Déplacez le dossier de l'application vers un emplacement où vous pouvez écrire : %w", e)
	}
	unlock, err := lockDataDir(dataDir)
	if errors.Is(err, errAlreadyOpen) {
		log.Print(alreadyOpenMessage)
		message(alreadyOpenMessage)
		return nil
	}
	if err != nil {
		return err
	}
	defer unlock()
	store, e := storage.Open(filepath.Join(dataDir, "state.db"))
	if e != nil {
		return e
	}
	defer store.Close()
	vault, e := auth.New(filepath.Join(dataDir, "credentials"))
	if e != nil {
		return e
	}
	tempRoot := filepath.Join(dataDir, "temporary")
	if e = auth.PrivateDir(tempRoot); e != nil {
		return e
	}
	entries, _ := os.ReadDir(tempRoot)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "run-") {
			if e = os.RemoveAll(filepath.Join(tempRoot, entry.Name())); e != nil {
				return fmt.Errorf("Nettoyage des fichiers temporaires de connexion impossible : %w", e)
			}
		}
	}
	toolsDir := filepath.Join(root, "tools")
	runner := &downloader.Runner{Store: store, Vault: vault, DataDir: dataDir, ToolsDir: toolsDir, Logs: &logging.Rotator{Dir: filepath.Join(dataDir, "logs")}}
	manager := queue.New(store, runner)
	defer manager.Close()
	api := web.New(store, manager, vault, dataDir, toolsDir)
	state, e := manager.Snapshot()
	if e != nil {
		return e
	}
	if state.Settings.Destination == "" {
		state.Settings.Destination = filepath.Join(root, "downloads")
		if e = manager.SaveSettings(state.Settings); e != nil {
			return e
		}
	}
	index, _ := assets.ReadFile("frontend/index.html")
	index = []byte(api.Index(string(index)))
	files := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b []byte
		switch r.URL.Path {
		case "/", "/index.html":
			b = index
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case "/style.css":
			b, _ = assets.ReadFile("frontend/style.css")
			w.Header().Set("Content-Type", "text/css")
		case "/app.js":
			b, _ = assets.ReadFile("frontend/dist/app.js")
			w.Header().Set("Content-Type", "text/javascript")
		default:
			http.NotFound(w, r)
			return
		}
		w.Write(b)
	})
	handler := api.Assets(files)
	if *preview != "" {
		if !strings.HasPrefix(*preview, "127.0.0.1:") {
			return fmt.Errorf("Preview must bind 127.0.0.1")
		}
		fmt.Println("UI preview http://" + *preview)
		return http.ListenAndServe(*preview, handler)
	}
	var app *application.App
	var window *application.WebviewWindow
	var quitting atomic.Bool
	app = application.New(application.Options{Name: "dl-manager", Description: "Gestionnaire de téléchargements", Icon: icon(), Assets: application.AssetOptions{Handler: handler}, Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: false, WebviewUserDataPath: filepath.Join(dataDir, "webview")}, OnShutdown: func() { quitting.Store(true); manager.Close() }})
	window = app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "dl-manager", Width: 1280, Height: 850, MinWidth: 840, MinHeight: 620, URL: "/", BackgroundColour: application.NewRGB(17, 21, 29)})
	quit := func() { quitting.Store(true); app.Quit() }
	api.Hide = func() { window.Minimise() }
	if runtime.GOOS == "windows" {
		window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
			if !quitting.Load() {
				e.Cancel()
				go quit()
			}
		})
		menu := application.NewMenu()
		menu.Add("Afficher dl-manager").OnClick(func(*application.Context) { window.Show(); window.Focus() })
		menu.Add("Tout mettre en pause").OnClick(func(*application.Context) { manager.Pause() })
		menu.AddSeparator()
		menu.Add("Quitter").OnClick(func(*application.Context) { quit() })
		tray := app.SystemTray.New()
		tray.SetIcon(icon())
		tray.SetTooltip("dl-manager - téléchargements")
		tray.SetMenu(menu)
		tray.OnClick(func() { window.Show(); window.Focus() })
		api.Hide = func() { window.Hide() }
	}
	api.Quit = quit
	api.OpenURL = func(url string) error { return app.Browser.OpenURL(url) }
	api.Pick = func(kind string) (string, error) {
		if kind == "backup-export" {
			return app.Dialog.SaveFile().AttachToWindow(window).AddFilter("Sauvegarde chiffrée", "*.dlmbackup").SetFilename("dl-manager-sites.dlmbackup").CanCreateDirectories(true).PromptForSingleSelection()
		}
		d := app.Dialog.OpenFile().AttachToWindow(window)
		if kind == "directory" {
			d.CanChooseDirectories(true).CanChooseFiles(false).CanCreateDirectories(true).SetTitle("Choisir un dossier de téléchargement")
		} else if kind == "backup-import" {
			d.AddFilter("Sauvegarde chiffrée", "*.dlmbackup").SetTitle("Importer les sites et comptes")
		} else if kind == "cookies" {
			d.AddFilter("Cookies Netscape", "*.txt").SetTitle("Importer cookies.txt")
		} else {
			d.SetTitle("Choisir un moteur")
			if runtime.GOOS == "windows" {
				d.AddFilter("Exécutable Windows", "*.exe")
			}
		}
		return d.PromptForSingleSelection()
	}
	return app.Run()
}
func icon() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 4; y < 60; y++ {
		for x := 4; x < 60; x++ {
			img.Set(x, y, color.RGBA{119, 217, 178, 255})
		}
	}
	dark := color.RGBA{17, 45, 33, 255}
	for y := 14; y < 39; y++ {
		for x := 28; x < 36; x++ {
			img.Set(x, y, dark)
		}
	}
	for y := 30; y < 46; y++ {
		for x := 16; x < 48; x++ {
			if x >= 16+(y-30) && x < 48-(y-30) {
				img.Set(x, y, dark)
			}
		}
	}
	for y := 49; y < 53; y++ {
		for x := 17; x < 47; x++ {
			img.Set(x, y, dark)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

var _ = model.Now
