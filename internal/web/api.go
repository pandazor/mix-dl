package web

import (
	"dl-manager/internal/downloader"
	"dl-manager/internal/model"
	"dl-manager/internal/platform"
	"dl-manager/internal/queue"
	"dl-manager/internal/storage"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

type CredentialVault interface {
	Save(string, []byte) error
	Load(string) ([]byte, error)
	Delete(string) error
}

type API struct {
	Store    *storage.Store
	Manager  *queue.Manager
	Vault    CredentialVault
	DataDir  string
	ToolsDir string
	Token    string
	Pick     func(string) (string, error)
	Hide     func()
	Quit     func()
	OpenURL  func(string) error
}

func New(s *storage.Store, m *queue.Manager, v CredentialVault, data, tools string) *API {
	return &API{Store: s, Manager: m, Vault: v, DataDir: data, ToolsDir: tools, Token: model.ID()}
}
func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, e error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(map[string]string{"error": e.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 12*1024*1024)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return errors.New("Corps JSON invalide")
	}
	return nil
}
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != "GET" && r.Header.Get("X-DL-Token") != a.Token {
		http.Error(w, "Invalid request token", 403)
		return
	}
	if r.Method == "POST" && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "JSON required", 415)
		return
	}
	var e error
	switch r.Method + " " + r.URL.Path {
	case "GET /api/state":
		state, err := a.Manager.Snapshot()
		if err != nil {
			e = err
			break
		}
		reply(w, map[string]any{"state": state, "sites": platform.Sites, "dataDir": a.DataDir, "version": "0.2.0", "platform": runtime.GOOS})
		return
	case "GET /api/profile":
		e = a.profileDetails(w, r)
		if e == nil {
			return
		}
	case "GET /api/logs":
		v, err := a.Store.Logs(r.URL.Query().Get("id"))
		if err != nil {
			e = err
			break
		}
		reply(w, v)
		return
	case "GET /api/engines":
		state, err := a.Manager.Snapshot()
		if err != nil {
			e = err
			break
		}
		out := []downloader.Engine{}
		for _, v := range []struct{ name, path string }{{"gallery-dl", state.Settings.GalleryPath}, {"yt-dlp", state.Settings.YTDLPPath}} {
			eng, err := downloader.Resolve(v.name, v.path, a.ToolsDir)
			if err != nil {
				eng = downloader.Engine{Name: v.name, Error: err.Error()}
			}
			out = append(out, eng)
		}
		reply(w, out)
		return
	case "POST /api/settings":
		var v model.Settings
		if e = decode(w, r, &v); e == nil {
			e = a.Manager.SaveSettings(v)
		}
	case "POST /api/queue/start":
		e = a.Manager.Start()
	case "POST /api/queue/pause":
		a.Manager.Pause()
	case "POST /api/quota/reset":
		e = a.Manager.ResetBudget()
	case "POST /api/jobs":
		var req struct {
			URLs        []string       `json:"urls"`
			Destination string         `json:"destination"`
			Layout      string         `json:"layout"`
			Filename    string         `json:"filename"`
			Filters     *model.Filters `json:"filters"`
			ProfileID   *string        `json:"profileId"`
		}
		if e = decode(w, r, &req); e != nil {
			break
		}
		if len(req.URLs) < 1 || len(req.URLs) > 500 {
			e = errors.New("Ajoutez entre 1 et 500 adresses")
			break
		}
		state, err := a.Manager.Snapshot()
		if err != nil {
			e = err
			break
		}
		added := 0
		problems := []string{}
		for _, raw := range req.URLs {
			raw = strings.TrimSpace(raw)
			site, err := model.ParseURL(raw)
			if err != nil {
				problems = append(problems, err.Error())
				continue
			}
			cfg := state.Settings
			sc := cfg.Sites[site]
			dest, layout, filename, filters, profile := cfg.Destination, cfg.Layout, cfg.Filename, cfg.Filters, sc.ProfileID
			if sc.Destination != "" {
				dest = sc.Destination
			}
			if sc.Layout != "" {
				layout = sc.Layout
			}
			if sc.Filename != "" {
				filename = sc.Filename
			}
			if sc.Filters != nil {
				filters = *sc.Filters
			}
			if req.Destination != "" {
				dest = req.Destination
			}
			if req.Layout != "" {
				layout = req.Layout
			}
			if req.Filename != "" {
				filename = req.Filename
			}
			if req.Filters != nil {
				filters = *req.Filters
			}
			profileMode := "inherit"
			if req.ProfileID != nil {
				profileMode = "explicit"
				if *req.ProfileID == "" {
					profileMode = "none"
				}
				profile = *req.ProfileID
			}
			if profile != "" {
				ok := false
				for _, p := range state.Profiles {
					if p.ID == profile && (p.Site == site || p.Site == "generic") {
						ok = true
					}
				}
				if !ok {
					problems = append(problems, "Le compte choisi ne correspond pas au site")
					continue
				}
			}
			j := model.Job{URL: raw, Site: site, Destination: dest, Layout: layout, Filename: filename, Filters: filters, ProfileID: profile, ProfileMode: profileMode}
			if err = a.Manager.Add(j); err != nil {
				problems = append(problems, err.Error())
			} else {
				added++
			}
		}
		reply(w, map[string]any{"added": added, "errors": problems})
		return
	case "POST /api/job/profile":
		var req struct {
			ID        string `json:"id"`
			Mode      string `json:"mode"`
			ProfileID string `json:"profileId"`
		}
		if e = decode(w, r, &req); e == nil {
			e = a.Manager.SetJobProfile(req.ID, req.Mode, req.ProfileID)
		}
	case "POST /api/job":
		var req struct {
			ID     string `json:"id"`
			Action string `json:"action"`
		}
		if e = decode(w, r, &req); e == nil {
			e = a.Manager.Action(req.ID, req.Action)
		}
	case "POST /api/history/clear":
		state, err := a.Manager.Snapshot()
		if err != nil {
			e = err
			break
		}
		for _, j := range state.Jobs {
			if j.Status == "completed" {
				if e = a.Manager.Action(j.ID, "delete"); e != nil {
					break
				}
			}
		}
	case "POST /api/profile":
		e = a.saveProfile(w, r)
	case "POST /api/profile/delete":
		var req struct {
			ID string `json:"id"`
		}
		if e = decode(w, r, &req); e != nil {
			break
		}
		e = a.Manager.WithProfileMutation(req.ID, func() error {
			if err := a.Vault.Delete(req.ID); err != nil {
				return err
			}
			if err := a.Vault.Delete(req.ID + "-cache"); err != nil {
				return err
			}
			return a.Store.DeleteProfile(req.ID)
		})
	case "POST /api/profile/validate":
		var req struct {
			ID string `json:"id"`
		}
		if e = decode(w, r, &req); e != nil {
			break
		}
		var b []byte
		b, e = a.Vault.Load(req.ID)
		if e == nil {
			var c downloader.Credentials
			e = json.Unmarshal(b, &c)
			if e == nil {
				reply(w, map[string]string{"message": "Profil déchiffrable. La connexion au site sera vérifiée lors du téléchargement."})
				return
			}
		}
	case "POST /api/pick":
		var req struct {
			Kind string `json:"kind"`
		}
		if e = decode(w, r, &req); e != nil {
			break
		}
		if req.Kind != "directory" && req.Kind != "cookies" && req.Kind != "engine" {
			e = errors.New("Type de sélection inconnu")
			break
		}
		if a.Pick == nil {
			e = errors.New("Sélection native disponible dans l'application")
			break
		}
		var path string
		path, e = a.Pick(req.Kind)
		if e == nil {
			reply(w, map[string]string{"path": path})
			return
		}
	case "POST /api/help":
		var req struct {
			Site string `json:"site"`
		}
		if e = decode(w, r, &req); e != nil {
			break
		}
		if a.OpenURL != nil {
			e = a.OpenURL(platform.Get(req.Site).URL)
		}
	case "POST /api/window/hide":
		if a.Hide != nil {
			a.Hide()
		}
	case "POST /api/window/quit":
		if a.Quit != nil {
			go a.Quit()
		}
	default:
		http.NotFound(w, r)
		return
	}
	if e != nil {
		fail(w, e)
		return
	}
	reply(w, map[string]bool{"ok": true})
}
func (a *API) Assets(files http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			a.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; object-src 'none'; frame-src 'none'; base-uri 'none'")
		files.ServeHTTP(w, r)
	})
}
func (a *API) Index(raw string) string { return strings.ReplaceAll(raw, "__DLM_TOKEN__", a.Token) }

var _ = filepath.Separator
