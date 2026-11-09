package web

import (
	"dl-manager/internal/auth"
	"dl-manager/internal/downloader"
	"dl-manager/internal/model"
	"dl-manager/internal/platform"
	"dl-manager/internal/queue"
	"dl-manager/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type API struct {
	Store    *storage.Store
	Manager  *queue.Manager
	Vault    *auth.Vault
	DataDir  string
	ToolsDir string
	Token    string
	Pick     func(string) (string, error)
	Hide     func()
	Quit     func()
	OpenURL  func(string) error
}

func New(s *storage.Store, m *queue.Manager, v *auth.Vault, data, tools string) *API {
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
		reply(w, map[string]any{"state": state, "sites": platform.Sites, "dataDir": a.DataDir, "version": "0.1.0"})
		return
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
			if req.ProfileID != nil {
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
			j := model.Job{URL: raw, Site: site, Destination: dest, Layout: layout, Filename: filename, Filters: filters, ProfileID: profile}
			if err = a.Manager.Add(j); err != nil {
				problems = append(problems, err.Error())
			} else {
				added++
			}
		}
		reply(w, map[string]any{"added": added, "errors": problems})
		return
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
			e = errors.New("Sélection native disponible dans l'application Windows")
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
func (a *API) saveProfile(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Site        string `json:"site"`
		Method      string `json:"method"`
		UserID      string `json:"userId"`
		Token       string `json:"token"`
		TokenSecret string `json:"tokenSecret"`
		CookiePath  string `json:"cookiePath"`
	}
	if e := decode(w, r, &req); e != nil {
		return e
	}
	if strings.TrimSpace(req.Name) == "" || len(req.Name) > 100 {
		return errors.New("Nom de compte obligatoire, 100 caractères maximum")
	}
	site := platform.Get(req.Site)
	if site.ID != req.Site {
		return errors.New("Site inconnu")
	}
	methodOK := false
	for _, m := range site.Methods {
		if m == req.Method {
			methodOK = true
		}
	}
	if !methodOK {
		return errors.New("Méthode de connexion incompatible avec ce site")
	}
	if req.ID == "" {
		req.ID = model.ID()
	}
	if a.Manager.ProfileBusy(req.ID) {
		return errors.New("Mettez en pause les tâches utilisant ce compte")
	}
	c := downloader.Credentials{Method: req.Method, UserID: strings.TrimSpace(req.UserID), Token: strings.TrimSpace(req.Token), TokenSecret: strings.TrimSpace(req.TokenSecret)}
	if c.Method == "cookies" {
		if req.CookiePath == "" {
			return errors.New("Sélectionnez un fichier cookies.txt")
		}
		f, e := os.Open(req.CookiePath)
		if e != nil {
			return errors.New("Fichier de cookies inaccessible")
		}
		defer f.Close()
		b, e := io.ReadAll(io.LimitReader(f, 10*1024*1024+1))
		if e != nil {
			return e
		}
		c.Cookies = string(b)
		if e = auth.ValidateCookies(c.Cookies, req.Site); e != nil {
			return e
		}
	} else {
		if c.Token == "" || len(c.Token) > 16384 {
			return errors.New("Token manquant ou trop long")
		}
		if c.Method == "api" {
			for _, ch := range c.UserID {
				if ch < '0' || ch > '9' {
					return errors.New("User ID doit être numérique")
				}
			}
			if c.UserID == "" {
				return errors.New("User ID obligatoire")
			}
		}
		if c.Method == "oauth1" && c.TokenSecret == "" {
			return errors.New("OAuth Token Secret obligatoire")
		}
	}
	payload, e := json.Marshal(c)
	if e != nil {
		return e
	}
	return a.Manager.WithProfileMutation(req.ID, func() error {
		if err := a.Vault.Save(req.ID, payload); err != nil {
			return fmt.Errorf("Chiffrement Windows indisponible : le compte n'a pas été enregistré. Lancez l'application dans votre session Windows habituelle. (%v)", err)
		}
		if err := a.Vault.Delete(req.ID + "-cache"); err != nil {
			return err
		}
		return a.Store.PutProfile(model.Profile{ID: req.ID, Name: req.Name, Site: req.Site, Method: req.Method, State: "configured"})
	})
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
