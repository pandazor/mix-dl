package web

import (
	"dl-manager/internal/downloader"
	"errors"
	"net/http"
	"strings"
)

func (a *API) Close() { a.OAuth.Close() }
func (a *API) oauth(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Site string `json:"site"`
		ID   string `json:"id"`
	}
	if decode(w, r, &req) != nil {
		return errors.New("Demande de connexion invalide")
	}
	switch strings.TrimPrefix(r.URL.Path, "/api/oauth/") {
	case "start":
		if req.Site != "tumblr" && req.Site != "deviantart" {
			return errors.New("Connexion guidée disponible pour Tumblr et DeviantArt")
		}
		state, e := a.Manager.Snapshot()
		if e != nil {
			return e
		}
		eng, e := downloader.Resolve("gallery-dl", state.Settings.GalleryPath, a.ToolsDir)
		if e != nil {
			return e
		}
		id, e := a.OAuth.Start(req.Site, eng)
		if e != nil {
			return e
		}
		reply(w, map[string]string{"id": id})
	case "status":
		v, e := a.OAuth.Status(req.ID)
		if e != nil {
			return e
		}
		reply(w, v)
	case "result":
		v, e := a.OAuth.Result(req.ID)
		if e != nil {
			return e
		}
		reply(w, v)
	case "cancel":
		if e := a.OAuth.Cancel(req.ID); e != nil {
			return e
		}
		reply(w, map[string]bool{"ok": true})
	case "open":
		v, e := a.OAuth.Status(req.ID)
		if e != nil {
			return e
		}
		if v.URL == "" {
			return errors.New("Page de connexion indisponible")
		}
		if a.OpenURL == nil {
			return errors.New("Utilisez le lien de connexion affiché")
		}
		if e = a.OpenURL(v.URL); e != nil {
			return errors.New("Ouverture du navigateur impossible ; utilisez le lien affiché")
		}
		reply(w, map[string]bool{"ok": true})
	}
	return nil
}
