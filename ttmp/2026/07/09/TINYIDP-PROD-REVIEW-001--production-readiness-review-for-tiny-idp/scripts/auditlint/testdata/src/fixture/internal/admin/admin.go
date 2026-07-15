package admin

import (
	"context"
	"database/sql"
)

func create(ctx context.Context, db *sql.DB) error { // want `persistence function create performs 2 mutation operations without Begin/BeginTx`
	if _, err := db.ExecContext(ctx, "INSERT INTO a VALUES (1)"); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, "INSERT INTO b VALUES (1)")
	return err
}

type transactionStore struct{}

func (*transactionStore) MaybeBeginTx() error              { return nil }
func (*transactionStore) CreateAccessTokenSession() error  { return nil }
func (*transactionStore) CreateRefreshTokenSession() error { return nil }
func (*transactionStore) CreateDeviceGrant() error         { return nil }

func tokenWrites(tx *transactionStore) error {
	if err := tx.MaybeBeginTx(); err != nil {
		return err
	}
	if err := tx.CreateAccessTokenSession(); err != nil {
		return err
	}
	return tx.CreateRefreshTokenSession()
}

func retrySingleStatementWrite(tx *transactionStore) error {
	for attempt := 0; attempt < 3; attempt++ {
		if err := tx.CreateDeviceGrant(); err != nil {
			continue
		}
		return nil
	}
	return nil
}
