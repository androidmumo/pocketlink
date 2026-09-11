CREATE TABLE devices (
 id TEXT PRIMARY KEY,
 sn TEXT NOT NULL UNIQUE,
 name TEXT NOT NULL,
 credential_hash TEXT NOT NULL UNIQUE,
 created_at INTEGER NOT NULL,
 revoked_at INTEGER
);
CREATE TABLE pairings (
 code_hash TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 expires_at INTEGER NOT NULL
);
