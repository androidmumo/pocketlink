ALTER TABLE invitations ADD COLUMN encrypted_code BLOB;
ALTER TABLE firmware_releases ADD COLUMN published_at INTEGER;
