package queue

import (
	"dl-manager/internal/model"
	"errors"
)

// SetJobProfile changes a paused/pending job without starting it.
func (m *Manager) SetJobProfile(id, mode, profile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("Application en cours de fermeture")
	}
	if _, ok := m.active[id]; ok {
		return errors.New("Mettez la tâche en pause avant de changer son compte")
	}
	s, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	for _, j := range s.Jobs {
		if j.ID == id {
			j.ProfileMode = mode
			j.ProfileID = profile
			if _, e = model.ResolveProfile(j, s); e != nil {
				return e
			}
			j.Status = "paused"
			j.Error = ""
			return m.store.PutJob(j)
		}
	}
	return errors.New("Tâche introuvable")
}
