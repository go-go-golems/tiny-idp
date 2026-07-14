package main

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
)

type message struct {
	ID            int64
	AuthorSubject string
	AuthorName    string
	Body          string
	CreatedAt     time.Time
}

type messageCursor struct {
	CreatedAt time.Time
	ID        int64
}

func (s *appStore) createMessage(ctx context.Context, value message) (message, error) {
	value.Body = strings.ReplaceAll(strings.ReplaceAll(value.Body, "\r\n", "\n"), "\r", "\n")
	if strings.TrimSpace(value.AuthorSubject) == "" || len(value.AuthorSubject) > 512 ||
		strings.TrimSpace(value.AuthorName) == "" || len(value.AuthorName) > 256 {
		return message{}, errors.New("message author is invalid")
	}
	if strings.TrimSpace(value.Body) == "" || len(value.Body) > 4096 || utf8.RuneCountInString(value.Body) > 1000 {
		return message{}, errors.New("message body must contain 1 to 1000 characters and at most 4096 bytes")
	}
	if value.CreatedAt.IsZero() {
		return message{}, errors.New("message creation time is required")
	}
	result, err := s.db.ExecContext(ctx, `
INSERT INTO messages(author_subject, author_name, body, created_at)
VALUES(?, ?, ?, ?)`, value.AuthorSubject, value.AuthorName, value.Body, formatAppTime(value.CreatedAt))
	if err != nil {
		return message{}, errors.Wrap(err, "create message")
	}
	value.ID, err = result.LastInsertId()
	if err != nil {
		return message{}, errors.Wrap(err, "read created message id")
	}
	return value, nil
}

func (s *appStore) listMessages(ctx context.Context, before *messageCursor, limit int) ([]message, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New("message page limit must be between 1 and 100")
	}
	query := `
SELECT id, author_subject, author_name, body, created_at
FROM messages
WHERE deleted_at IS NULL`
	args := make([]any, 0, 3)
	if before != nil {
		if before.CreatedAt.IsZero() || before.ID < 1 {
			return nil, errors.New("message cursor is invalid")
		}
		stamp := formatAppTime(before.CreatedAt)
		query += " AND (created_at < ? OR (created_at = ? AND id < ?))"
		args = append(args, stamp, stamp, before.ID)
	}
	query += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "list messages")
	}
	defer rows.Close()
	values := make([]message, 0, limit)
	for rows.Next() {
		var value message
		var createdAt string
		if err := rows.Scan(&value.ID, &value.AuthorSubject, &value.AuthorName, &value.Body, &createdAt); err != nil {
			return nil, errors.Wrap(err, "scan message")
		}
		if value.CreatedAt, err = parseAppTime(createdAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterate messages")
	}
	return values, nil
}
