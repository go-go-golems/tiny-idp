CREATE UNIQUE INDEX idx_signing_keys_single_active
ON signing_keys(active)
WHERE active = 1;
