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
