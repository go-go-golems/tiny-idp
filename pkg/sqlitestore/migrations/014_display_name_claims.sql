CREATE TABLE display_name_claims (
    normalized_name TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE
);
