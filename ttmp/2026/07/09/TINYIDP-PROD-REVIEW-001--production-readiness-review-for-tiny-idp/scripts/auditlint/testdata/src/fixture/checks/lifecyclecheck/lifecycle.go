package lifecyclecheck

type database struct{}

func (*database) ExecContext() error { return nil }

type sqlFositeStore struct{ db *database }

func (s *sqlFositeStore) CreateAccessTokenSession() error { // want "must use tokenExec"
	return s.db.ExecContext()
}

func (s *sqlFositeStore) CreateRefreshTokenSession() error {
	return s.tokenExec()
}

func (*sqlFositeStore) tokenExec() error { return nil }
