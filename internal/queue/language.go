package queue

import (
	"dl-manager/internal/model"
	"errors"
)

func (m *Manager) SaveLanguage(language string) error {
	if language == "" || !model.ValidLanguage(language) {
		return errors.New("Langue non prise en charge")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("Application en cours de fermeture")
	}
	s, e := m.store.Snapshot()
	if e != nil {
		return e
	}
	s.Settings.Language = language
	return m.store.SaveSettings(s.Settings)
}
