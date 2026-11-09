package web

import (
	"bytes"
	"dl-manager/internal/auth"
	"dl-manager/internal/backup"
	"dl-manager/internal/downloader"
	"dl-manager/internal/model"
	"dl-manager/internal/platform"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type backupAccount struct {
	Profile     model.Profile          `json:"profile"`
	Credentials downloader.Credentials `json:"credentials"`
}
type backupDocument struct {
	CustomSites map[string]model.CustomSite   `json:"customSites,omitempty"`
	Version     int                           `json:"version"`
	Sites       map[string]model.SiteSettings `json:"sites"`
	Accounts    []backupAccount               `json:"accounts"`
}

var backupID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)
var invalidBackup = errors.New("Contenu de sauvegarde invalide ou version non prise en charge")

// validateBackup rejects all malformed entries before the vault is modified.
func validateBackup(doc *backupDocument) (int, error) {
	if (doc.Version != 1 && doc.Version != 2) || (doc.Version == 1 && len(doc.CustomSites) > 0) || model.ValidateCustomSites(doc.CustomSites) != nil || len(doc.Accounts) > 1000 || len(doc.Sites) > len(platform.Sites)+len(doc.CustomSites) {
		return 0, invalidBackup
	}
	definition := model.DefaultSettings()
	definition.CustomSites = doc.CustomSites
	profiles := map[string]model.Profile{}
	for _, entry := range doc.Accounts {
		p, c := entry.Profile, entry.Credentials
		if !backupID.MatchString(p.ID) || profiles[p.ID].ID != "" || strings.TrimSpace(p.Name) == "" || len(p.Name) > 100 || platform.GetForSettings(p.Site, definition).ID != p.Site || p.Method != c.Method {
			return 0, invalidBackup
		}
		methodOK := false
		for _, m := range platform.GetForSettings(p.Site, definition).Methods {
			if m == p.Method {
				methodOK = true
			}
		}
		if !methodOK {
			return 0, invalidBackup
		}
		if len(c.Token) > 16384 || len(c.TokenSecret) > 16384 || len(c.UserID) > 100 || len(c.Cookies) > 10*1024*1024 {
			return 0, invalidBackup
		}
		if c.Method == "cookies" {
			var err error
			if custom, ok := doc.CustomSites[p.Site]; ok {
				err = auth.ValidateCookiesForDomain(c.Cookies, custom.Domain)
			} else {
				err = auth.ValidateCookies(c.Cookies, p.Site)
			}
			if err != nil {
				return 0, invalidBackup
			}
		} else {
			if c.Token == "" {
				return 0, invalidBackup
			}
			if c.Method == "api" {
				if c.UserID == "" {
					return 0, invalidBackup
				}
				for _, ch := range c.UserID {
					if ch < '0' || ch > '9' {
						return 0, invalidBackup
					}
				}
			}
			if c.Method == "oauth1" && c.TokenSecret == "" {
				return 0, invalidBackup
			}
		}
		profiles[p.ID] = p
	}
	reset := 0
	cfg := model.DefaultSettings()
	cfg.Sites = doc.Sites
	cfg.CustomSites = doc.CustomSites
	for id, site := range doc.Sites {
		if platform.GetForSettings(id, definition).ID != id || len(site.Destination) > 8192 || strings.ContainsRune(site.Destination, 0) {
			return 0, invalidBackup
		}
		if site.ProfileID != "" {
			p, ok := profiles[site.ProfileID]
			if !ok || (p.Site != id && p.Site != "generic") {
				return 0, invalidBackup
			}
		}
		// Foreign-OS paths become inheritance rather than invalid destinations.
		if site.Destination != "" && !filepath.IsAbs(site.Destination) {
			site.Destination = ""
			doc.Sites[id] = site
			reset++
		}
	}
	if cfg.Validate() != nil {
		return 0, invalidBackup
	}
	return reset, nil
}
func (a *API) exportBackup(w http.ResponseWriter, r *http.Request) error {
	a.backupMu.Lock()
	defer a.backupMu.Unlock()
	var req struct {
		Path         string `json:"path"`
		Passphrase   string `json:"passphrase"`
		Confirmation string `json:"confirmation"`
		Acknowledge  bool   `json:"acknowledge"`
	}
	if decode(w, r, &req) != nil {
		return errors.New("Demande d'export invalide")
	}
	if !req.Acknowledge {
		return errors.New("Confirmez avoir compris que la sauvegarde contient vos accès aux comptes")
	}
	if req.Passphrase != req.Confirmation {
		return errors.New("Les deux phrases secrètes ne correspondent pas")
	}
	if e := backup.ValidatePassphrase(req.Passphrase); e != nil {
		return e
	}
	if !filepath.IsAbs(req.Path) || !strings.EqualFold(filepath.Ext(req.Path), ".dlmbackup") {
		return errors.New("Choisissez un nouveau fichier .dlmbackup avec un chemin absolu")
	}
	var doc backupDocument
	e := a.Manager.WithConfigurationRead(func() error {
		state, e := a.Store.Snapshot()
		if e != nil {
			return errors.New("Impossible de lire les réglages")
		}
		doc = backupDocument{Version: 1, Sites: state.Settings.Sites, Accounts: []backupAccount{}, CustomSites: state.Settings.CustomSites}
		if len(doc.CustomSites) > 0 {
			doc.Version = 2
		}
		total := 0
		for _, p := range state.Profiles {
			raw, e := a.Vault.Load(p.ID)
			if e != nil {
				return errors.New("Export impossible : coffre d'un compte inaccessible")
			}
			total += len(raw)
			if total > backup.MaxSize {
				clear(raw)
				return errors.New("Sauvegarde trop volumineuse (32 Mio maximum)")
			}
			var c downloader.Credentials
			e = json.Unmarshal(raw, &c)
			clear(raw)
			if e != nil {
				return errors.New("Export impossible : compte enregistré invalide")
			}
			doc.Accounts = append(doc.Accounts, backupAccount{p, c})
		}
		_, e = validateBackup(&doc)
		return e
	})
	if e != nil {
		return e
	}
	plain, e := json.Marshal(doc)
	if e != nil {
		return invalidBackup
	}
	defer clear(plain)
	encrypted, e := backup.Encrypt(plain, req.Passphrase)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(req.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return errors.New("Fichier déjà existant ou dossier inaccessible : choisissez un nouveau nom")
	}
	complete := false
	defer func() {
		f.Close()
		if !complete {
			os.Remove(req.Path)
		}
	}()
	if _, e = f.Write(encrypted); e != nil {
		return errors.New("Écriture de la sauvegarde impossible")
	}
	if e = f.Sync(); e != nil {
		return errors.New("Enregistrement de la sauvegarde impossible")
	}
	if e = f.Close(); e != nil {
		return errors.New("Fermeture de la sauvegarde impossible")
	}
	complete = true
	reply(w, map[string]any{"ok": true, "accounts": len(doc.Accounts), "sites": len(doc.Sites)})
	return nil
}
func (a *API) importBackup(w http.ResponseWriter, r *http.Request) error {
	a.backupMu.Lock()
	defer a.backupMu.Unlock()
	var req struct {
		Path         string `json:"path"`
		Passphrase   string `json:"passphrase"`
		ReplaceSites bool   `json:"replaceSites"`
	}
	if decode(w, r, &req) != nil {
		return errors.New("Demande d'import invalide")
	}
	if e := backup.ValidatePassphrase(req.Passphrase); e != nil {
		return e
	}
	if !filepath.IsAbs(req.Path) {
		return errors.New("Choisissez une sauvegarde avec un chemin absolu")
	}
	f, e := os.Open(req.Path)
	if e != nil {
		return errors.New("Sauvegarde inaccessible")
	}
	blob, e := io.ReadAll(io.LimitReader(f, backup.MaxSize+1))
	f.Close()
	if e != nil {
		return errors.New("Lecture de la sauvegarde impossible")
	}
	plain, e := backup.Decrypt(blob, req.Passphrase)
	if e != nil {
		return e
	}
	defer clear(plain)
	var doc backupDocument
	decoder := json.NewDecoder(bytes.NewReader(plain))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&doc) != nil {
		return invalidBackup
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return invalidBackup
	}
	reset, e := validateBackup(&doc)
	if e != nil {
		return e
	}
	importedSites := 0
	e = a.Manager.WithIdle(func() (result error) {
		state, e := a.Store.Snapshot()
		if e != nil {
			return errors.New("Impossible de lire les réglages")
		}
		siteRemap := map[string]string{}
		preserveCustom := map[string]bool{}
		if state.Settings.CustomSites == nil {
			state.Settings.CustomSites = map[string]model.CustomSite{}
		}
		for old, incoming := range doc.CustomSites {
			target := ""
			for id, current := range state.Settings.CustomSites {
				if current.Domain == incoming.Domain {
					target = id
					preserveCustom[id] = true
					break
				}
			}
			if target == "" {
				for {
					target = "custom_" + model.ID()
					if _, exists := state.Settings.CustomSites[target]; !exists {
						break
					}
				}
				state.Settings.CustomSites[target] = incoming
			} else if req.ReplaceSites {
				state.Settings.CustomSites[target] = incoming
			}
			siteRemap[old] = target
		}
		if e = state.Settings.Validate(); e != nil {
			return invalidBackup
		}
		remap := map[string]string{}
		profiles := []model.Profile{}
		written := []string{}
		committed := false
		used := map[string]bool{}
		for _, p := range state.Profiles {
			used[p.ID] = true
		}
		defer func() {
			if !committed {
				for _, id := range written {
					if a.Vault.Delete(id) != nil {
						result = errors.Join(result, errors.New("Nettoyage des nouveaux comptes incomplet ; les comptes existants sont conservés"))
					}
				}
			}
		}()
		for _, entry := range doc.Accounts {
			p := entry.Profile
			if target, ok := siteRemap[p.Site]; ok {
				p.Site = target
			}
			old := p.ID
			for {
				p.ID = model.ID()
				if !used[p.ID] {
					break
				}
			}
			used[p.ID] = true
			p.State = "configured"
			remap[old] = p.ID
			raw, e := json.Marshal(entry.Credentials)
			if e != nil {
				return invalidBackup
			}
			written = append(written, p.ID)
			e = a.Vault.Save(p.ID, raw)
			if e != nil {
				clear(raw)
				return errors.New("Import annulé : coffre local indisponible")
			}
			stored, e := a.Vault.Load(p.ID)
			valid := e == nil && bytes.Equal(raw, stored)
			clear(raw)
			clear(stored)
			if !valid {
				return errors.New("Import annulé : vérification du coffre local impossible")
			}
			profiles = append(profiles, p)
		}
		for id, site := range doc.Sites {
			if target, ok := siteRemap[id]; ok {
				id = target
			}
			if _, exists := state.Settings.Sites[id]; (exists || preserveCustom[id]) && !req.ReplaceSites {
				continue
			}
			if site.ProfileID != "" {
				site.ProfileID = remap[site.ProfileID]
			}
			state.Settings.Sites[id] = site
			importedSites++
		}
		if e = a.Store.ImportSites(state.Settings, profiles); e != nil {
			return errors.New("Import annulé : enregistrement des comptes et réglages impossible")
		}
		committed = true
		return nil
	})
	if e != nil {
		return e
	}
	reply(w, map[string]any{"ok": true, "accounts": len(doc.Accounts), "sites": importedSites, "resetPaths": reset})
	return nil
}
