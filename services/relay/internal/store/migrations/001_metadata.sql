CREATE TABLE app_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO app_metadata(key, value) VALUES ('protocol_version', '1');
