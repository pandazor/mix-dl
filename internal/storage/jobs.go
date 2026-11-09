package storage

import "dl-manager/internal/model"

func (s *Store) PutJobs(jobs []model.Job) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, j := range jobs {
		b, e := encode(j)
		if e != nil {
			return e
		}
		if _, e = tx.Exec("UPDATE jobs SET data=? WHERE id=?", b, j.ID); e != nil {
			return e
		}
	}
	return tx.Commit()
}
