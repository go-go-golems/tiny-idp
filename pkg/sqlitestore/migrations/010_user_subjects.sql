ALTER TABLE users ADD COLUMN subject TEXT;
UPDATE users SET subject = COALESCE(json_extract(data, '$.Sub'), '');
CREATE UNIQUE INDEX idx_users_subject ON users(subject);
