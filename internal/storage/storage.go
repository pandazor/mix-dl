package storage

import (
	"database/sql"
	"dl-manager/internal/model"
	"encoding/json"
	"errors"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Store{db}
	_, e = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS kv (key TEXT PRIMARY KEY,value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS jobs (id TEXT PRIMARY KEY,ord INTEGER NOT NULL,data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS profiles(id TEXT PRIMARY KEY,data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS logs(id INTEGER PRIMARY KEY AUTOINCREMENT,job TEXT NOT NULL,at TEXT NOT NULL,line TEXT NOT NULL);
 CREATE INDEX IF NOT EXISTS logs_job ON logs(job,id);
 INSERT OR IGNORE INTO kv(key,value) VALUES('usage','0');`)
	if e != nil {
		db.Close()
		return nil, e
	}
	snapshot, e := s.Snapshot()
	if e != nil {
		db.Close()
		return nil, e
	}
	for _, j := range snapshot.Jobs {
		if j.Status == "running" {
			j.Status = "interrupted"
			j.Error = "Interrompu à la fermeture. Cliquez sur Reprendre."
			if e = s.PutJob(j); e != nil {
				db.Close()
				return nil, e
			}
		}
	}
	return s, nil
}
func (s *Store) Close() error      { return s.db.Close() }
func encode(v any) (string, error) { b, e := json.Marshal(v); return string(b), e }
func (s *Store) PutJob(j model.Job) error {
	b, e := encode(j)
	if e != nil {
		return e
	}
	_, e = s.db.Exec("INSERT INTO jobs(id,ord,data) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET ord=excluded.ord,data=excluded.data", j.ID, j.Order, b)
	return e
}
func (s *Store) DeleteJob(id string) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec("DELETE FROM logs WHERE job=?", id); e != nil {
		return e
	}
	if _, e = tx.Exec("DELETE FROM jobs WHERE id=?", id); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) SaveSettings(v model.Settings) error {
	if e := v.Validate(); e != nil {
		return e
	}
	b, e := encode(v)
	if e != nil {
		return e
	}
	_, e = s.db.Exec("INSERT INTO kv VALUES('settings',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", b)
	return e
}
func (s *Store) PutProfile(v model.Profile) error {
	b, e := encode(v)
	if e != nil {
		return e
	}
	_, e = s.db.Exec("INSERT INTO profiles VALUES(?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", v.ID, b)
	return e
}

// PutProfileWithDefault commits account metadata and its default selection together.
func (s *Store) PutProfileWithDefault(v model.Profile, makeDefault bool) error {
	if makeDefault {
		return s.PutProfileSelection(v, &makeDefault)
	}
	return s.PutProfileSelection(v, nil)
}

// A nil selection preserves the default; false clears only this account's selection.
func (s *Store) PutProfileSelection(v model.Profile, selection *bool) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	raw, e := encode(v)
	if e != nil {
		return e
	}
	if _, e = tx.Exec("INSERT INTO profiles VALUES(?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", v.ID, raw); e != nil {
		return e
	}
	if selection != nil {
		cfg := model.DefaultSettings()
		e = tx.QueryRow("SELECT value FROM kv WHERE key='settings'").Scan(&raw)
		if e == nil {
			if e = json.Unmarshal([]byte(raw), &cfg); e != nil {
				return e
			}
		} else if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if cfg.Sites == nil {
			cfg.Sites = map[string]model.SiteSettings{}
		}
		site := cfg.Sites[v.Site]
		if *selection {
			site.ProfileID = v.ID
		} else if site.ProfileID == v.ID {
			site.ProfileID = ""
		}
		cfg.Sites[v.Site] = site
		raw, e = encode(cfg)
		if e != nil {
			return e
		}
		if _, e = tx.Exec("INSERT INTO kv VALUES('settings',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", raw); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (s *Store) DeleteProfile(id string) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var raw string
	e = tx.QueryRow("SELECT value FROM kv WHERE key='settings'").Scan(&raw)
	if e == nil {
		var settings model.Settings
		if e = json.Unmarshal([]byte(raw), &settings); e != nil {
			return e
		}
		for site, cfg := range settings.Sites {
			if cfg.ProfileID == id {
				cfg.ProfileID = ""
				settings.Sites[site] = cfg
			}
		}
		raw, e = encode(settings)
		if e != nil {
			return e
		}
		if _, e = tx.Exec("UPDATE kv SET value=? WHERE key='settings'", raw); e != nil {
			return e
		}
	} else if !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	if _, e = tx.Exec("DELETE FROM profiles WHERE id=?", id); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) AddUsage(n int64) error {
	if n < 0 {
		return errors.New("negative usage")
	}
	_, e := s.db.Exec("UPDATE kv SET value=CAST(value AS INTEGER)+? WHERE key='usage'", n)
	return e
}
func (s *Store) Progress(j model.Job, n int64) error {
	if n < 0 {
		return errors.New("negative usage")
	}
	b, e := encode(j)
	if e != nil {
		return e
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec("INSERT INTO jobs(id,ord,data) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET ord=excluded.ord,data=excluded.data", j.ID, j.Order, b); e != nil {
		return e
	}
	if _, e = tx.Exec("UPDATE kv SET value=CAST(value AS INTEGER)+? WHERE key='usage'", n); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) ResetBudget() error {
	_, e := s.db.Exec("UPDATE kv SET value='0' WHERE key='usage'")
	return e
}
func (s *Store) Log(id, line string) error {
	_, e := s.db.Exec("INSERT INTO logs(job,at,line) VALUES(?,?,?)", id, model.Now(), line)
	if e != nil {
		return e
	}
	_, e = s.db.Exec("DELETE FROM logs WHERE job=? AND id NOT IN (SELECT id FROM logs WHERE job=? ORDER BY id DESC LIMIT 500)", id, id)
	return e
}

type LogLine struct {
	At   string `json:"at"`
	Line string `json:"line"`
}

func (s *Store) Logs(id string) ([]LogLine, error) {
	rows, e := s.db.Query("SELECT at,line FROM logs WHERE job=? ORDER BY id", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []LogLine{}
	for rows.Next() {
		var l LogLine
		if e = rows.Scan(&l.At, &l.Line); e != nil {
			return nil, e
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
func (s *Store) Snapshot() (model.Snapshot, error) {
	out := model.Snapshot{Settings: model.DefaultSettings(), Jobs: []model.Job{}, Profiles: []model.Profile{}}
	tx, e := s.db.Begin()
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	var raw string
	e = tx.QueryRow("SELECT value FROM kv WHERE key='settings'").Scan(&raw)
	if e == nil {
		if e = json.Unmarshal([]byte(raw), &out.Settings); e != nil {
			return out, e
		}
	} else if !errors.Is(e, sql.ErrNoRows) {
		return out, e
	}
	if out.Settings.Sites == nil {
		out.Settings.Sites = map[string]model.SiteSettings{}
	}
	if e = tx.QueryRow("SELECT CAST(value AS INTEGER) FROM kv WHERE key='usage'").Scan(&out.UsedBytes); e != nil {
		return out, e
	}
	rows, e := tx.Query("SELECT data FROM jobs ORDER BY ord,id")
	if e != nil {
		return out, e
	}
	for rows.Next() {
		if e = rows.Scan(&raw); e != nil {
			rows.Close()
			return out, e
		}
		var j model.Job
		if e = json.Unmarshal([]byte(raw), &j); e != nil {
			rows.Close()
			return out, e
		}
		out.Jobs = append(out.Jobs, j)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	rows, e = tx.Query("SELECT data FROM profiles ORDER BY id")
	if e != nil {
		return out, e
	}
	for rows.Next() {
		if e = rows.Scan(&raw); e != nil {
			rows.Close()
			return out, e
		}
		var p model.Profile
		if e = json.Unmarshal([]byte(raw), &p); e != nil {
			rows.Close()
			return out, e
		}
		out.Profiles = append(out.Profiles, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	return out, tx.Commit()
}
