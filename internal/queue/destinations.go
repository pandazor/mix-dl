package queue

import (
	"dl-manager/internal/model"
	"errors"
	"path/filepath"
	"strings"
)

// WithIdle serializes a configuration transfer with dispatch and other mutations.
// The callback must use Store/Vault directly and must not re-enter Manager.
func (m *Manager) WithIdle(change func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.resetting {
		return errors.New("Application en cours de fermeture ou de remise à zéro")
	}
	if m.running || len(m.active) > 0 {
		return errors.New("Mettez la file en pause et attendez l'arrêt des tâches")
	}
	return change()
}
func (m *Manager) RefreshDestinations(ids []string) error {
	return m.WithIdle(func() error {
		if len(ids) == 0 || len(ids) > 500 {
			return errors.New("Sélectionnez entre 1 et 500 tâches")
		}
		s, e := m.store.Snapshot()
		if e != nil {
			return e
		}
		byID := map[string]model.Job{}
		for _, j := range s.Jobs {
			byID[j.ID] = j
		}
		updates := []model.Job{}
		seen := map[string]bool{}
		for _, id := range ids {
			j, ok := byID[id]
			if !ok || seen[id] {
				return errors.New("Sélection de tâches invalide")
			}
			seen[id] = true
			if !j.CanResetDestination() || j.DestinationMode == "explicit" {
				return errors.New("Seules les tâches jamais démarrées sans dossier personnalisé peuvent être actualisées")
			}
			j.DestinationMode = "inherit"
			j.Destination = model.SiteDestination(s.Settings, j.Site)
			if j.Destination == "" || !filepath.IsAbs(j.Destination) {
				return errors.New("Choisissez un dossier global ou un dossier propre au site")
			}
			updates = append(updates, j)
		}
		return m.store.PutJobs(updates)
	})
}

// SaveDestination updates only the global folder, preserving concurrent site/options edits.
func (m *Manager) SaveDestination(destination string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.resetting {
		return errors.New("Application en cours de fermeture ou de remise à zéro")
	}
	destination = strings.TrimSpace(destination)
	if destination == "" || !filepath.IsAbs(destination) || strings.ContainsRune(destination, 0) {
		return errors.New("Choisissez un dossier global avec un chemin absolu")
	}
	state, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	state.Settings.Destination = destination
	if e = m.store.SaveSettings(state.Settings); e != nil {
		return e
	}
	m.signal()
	return nil
}

// WithConfigurationRead captures settings and credentials coherently with their
// mutations. Active workers continue; encryption/file IO must happen after it.
// The callback must not re-enter Manager or mutate Store/Vault.
func (m *Manager) WithConfigurationRead(read func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.resetting {
		return errors.New("Application en cours de fermeture ou de remise à zéro")
	}
	return read()
}
