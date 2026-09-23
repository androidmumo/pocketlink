ALTER TABLE users ADD COLUMN disabled_at INTEGER;
ALTER TABLE users ADD COLUMN auth_version INTEGER NOT NULL DEFAULT 0;
CREATE TRIGGER users_admin_protected BEFORE UPDATE OF disabled_at,auth_version,password_salt,password_hash ON users WHEN OLD.id='admin' BEGIN SELECT RAISE(ABORT,'administrator uses file configuration'); END;
