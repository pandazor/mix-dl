package storage

import "dl-manager/internal/model"

// ImportSites publishes account metadata and site settings in one transaction.
func (s *Store) ImportSites(cfg model.Settings, profiles []model.Profile) error {
	if e := cfg.Validate(); e != nil {
		return e
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, p := range profiles {
		b, e := encode(p)
		if e != nil {
			return e
		}
		if _, e = tx.Exec("INSERT INTO profiles(id,data) VALUES(?,?)", p.ID, b); e != nil {
			return e
		}
	}
	b, e := encode(cfg)
	if e != nil {
		return e
	}
	if _, e = tx.Exec("INSERT INTO kv VALUES('settings',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", b); e != nil {
		return e
	}
	return tx.Commit()
}
