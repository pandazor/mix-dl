package web

import (
	"bytes"
	"dl-manager/internal/auth"
	"dl-manager/internal/downloader"
	"dl-manager/internal/model"
	"dl-manager/internal/platform"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// profileDetails exposes only public identifiers and presence flags, never secrets.
func (a *API) profileDetails(w http.ResponseWriter, r *http.Request) error {
	s, e := a.Store.Snapshot()
	if e != nil {
		return e
	}
	for _, p := range s.Profiles {
		if p.ID == r.URL.Query().Get("id") {
			b, e := a.Vault.Load(p.ID)
			if e != nil {
				return fmt.Errorf("Compte enregistré mais coffre inaccessible : %w", e)
			}
			var c downloader.Credentials
			if e = json.Unmarshal(b, &c); e != nil {
				return e
			}
			reply(w, map[string]any{"profile": p, "userId": c.UserID, "hasToken": c.Token != "", "hasTokenSecret": c.TokenSecret != "", "hasCookies": c.Cookies != ""})
			return nil
		}
	}
	return errors.New("Compte introuvable")
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
		SetDefault  *bool  `json:"setDefault"`
	}
	if e := decode(w, r, &req); e != nil {
		return e
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 100 {
		return errors.New("Nom de compte obligatoire, 100 caractères maximum")
	}
	site := platform.Get(req.Site)
	if site.ID != req.Site {
		return errors.New("Site inconnu")
	}
	valid := false
	for _, m := range site.Methods {
		if m == req.Method {
			valid = true
		}
	}
	if !valid {
		return errors.New("Méthode de connexion incompatible avec ce site")
	}
	creating := req.ID == ""
	if creating {
		req.ID = model.ID()
	}
	return a.Manager.WithProfileMutation(req.ID, func() error {
		state, e := a.Store.Snapshot()
		if e != nil {
			return e
		}
		c := downloader.Credentials{Method: req.Method}
		if !creating {
			found := false
			for _, p := range state.Profiles {
				if p.ID == req.ID {
					found = true
					if p.Site != req.Site {
						return errors.New("Le site d'un compte existant ne peut pas être changé")
					}
					if p.Method == req.Method {
						b, e := a.Vault.Load(req.ID)
						if e != nil {
							return fmt.Errorf("Impossible de relire le compte enregistré : %w", e)
						}
						if e = json.Unmarshal(b, &c); e != nil {
							return e
						}
					}
				}
			}
			if !found {
				return errors.New("Compte introuvable")
			}
		}
		if u := strings.TrimSpace(req.UserID); u != "" {
			c.UserID = u
		}
		if token := strings.TrimSpace(req.Token); token != "" {
			c.Token = token
		}
		if token := strings.TrimSpace(req.TokenSecret); token != "" {
			c.TokenSecret = token
		}
		if c.Method == "cookies" {
			if req.CookiePath != "" {
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
			}
			if e = auth.ValidateCookies(c.Cookies, req.Site); e != nil {
				return e
			}
		} else {
			if c.Token == "" || len(c.Token) > 16384 {
				return errors.New("Token manquant ou trop long")
			}
			if c.Method == "api" {
				if c.UserID == "" {
					return errors.New("User ID obligatoire")
				}
				for _, ch := range c.UserID {
					if ch < '0' || ch > '9' {
						return errors.New("User ID doit être numérique")
					}
				}
			}
			if c.Method == "oauth1" && (c.TokenSecret == "" || len(c.TokenSecret) > 16384) {
				return errors.New("OAuth Token Secret manquant ou trop long")
			}
		}
		payload, e := json.Marshal(c)
		if e != nil {
			return e
		}
		if e = a.Vault.Save(req.ID, payload); e != nil {
			return fmt.Errorf("Compte non enregistré : coffre de la session indisponible. %w", e)
		}
		// Confirm the stored bytes can be decrypted before publishing success to the UI.
		stored, e := a.Vault.Load(req.ID)
		if e != nil {
			return fmt.Errorf("Vérification du compte enregistré impossible : %w", e)
		}
		if !bytes.Equal(stored, payload) {
			return errors.New("La vérification du compte enregistré a échoué")
		}
		if e = a.Vault.Delete(req.ID + "-cache"); e != nil {
			return e
		}
		selection := req.SetDefault
		if selection == nil && creating && state.Settings.Sites[req.Site].ProfileID == "" {
			yes := true
			selection = &yes
		}
		return a.Store.PutProfileSelection(model.Profile{ID: req.ID, Name: req.Name, Site: req.Site, Method: req.Method, State: "configured"}, selection)
	})
}
