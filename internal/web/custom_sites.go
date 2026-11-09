package web

import (
	"net/http"
)

func (a *API) customSite(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Domain string `json:"domain"`
	}
	if e := decode(w, r, &req); e != nil {
		return e
	}
	if r.URL.Path == "/api/custom-site/delete" {
		if e := a.Manager.DeleteCustomSite(req.ID); e != nil {
			return e
		}
		reply(w, map[string]bool{"ok": true})
		return nil
	}
	id, e := a.Manager.SaveCustomSite(req.ID, req.Name, req.Domain)
	if e != nil {
		return e
	}
	reply(w, map[string]string{"id": id})
	return nil
}
