package queue

import (
	"dl-manager/internal/model"
	"errors"
	"strings"
)

func (m *Manager) SaveCustomSite(id, name, domain string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return "", errors.New("Application en cours de fermeture")
	}
	d, e := model.NormalizeDomain(domain)
	if e != nil {
		return "", e
	}
	state, e := m.store.Snapshot()
	if e != nil {
		return "", e
	}
	if state.Settings.CustomSites == nil {
		state.Settings.CustomSites = map[string]model.CustomSite{}
	}
	if id == "" {
		id = "custom_" + model.ID()
	} else {
		old, ok := state.Settings.CustomSites[id]
		if !ok {
			return "", errors.New("Site personnalisé introuvable")
		}
		if old.Domain != d {
			return "", errors.New("Le domaine d'un site existant ne peut pas être changé")
		}
	}
	state.Settings.CustomSites[id] = model.CustomSite{Name: strings.TrimSpace(name), Domain: d}
	if e = m.store.SaveSettings(state.Settings); e != nil {
		return "", e
	}
	return id, nil
}
func (m *Manager) DeleteCustomSite(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("Application en cours de fermeture")
	}
	state, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	if _, ok := state.Settings.CustomSites[id]; !ok {
		return errors.New("Site personnalisé introuvable")
	}
	for _, p := range state.Profiles {
		if p.Site == id {
			return errors.New("Retirez d'abord les comptes de ce site")
		}
	}
	for _, j := range state.Jobs {
		if j.Site == id {
			return errors.New("Retirez d'abord les tâches de ce site (y compris dans l'historique)")
		}
	}
	delete(state.Settings.CustomSites, id)
	delete(state.Settings.Sites, id)
	return m.store.SaveSettings(state.Settings)
}
