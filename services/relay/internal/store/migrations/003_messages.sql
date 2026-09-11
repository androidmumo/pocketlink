CREATE TABLE rooms(id TEXT PRIMARY KEY,name TEXT NOT NULL,created_at INTEGER NOT NULL,archived_at INTEGER);
CREATE TABLE room_members(room_id TEXT NOT NULL REFERENCES rooms(id),device_id TEXT NOT NULL REFERENCES devices(id),PRIMARY KEY(room_id,device_id));
CREATE TABLE messages(id INTEGER PRIMARY KEY AUTOINCREMENT,room_id TEXT NOT NULL REFERENCES rooms(id),request_id TEXT NOT NULL,body TEXT NOT NULL,created_at INTEGER NOT NULL,UNIQUE(room_id,request_id));
CREATE INDEX messages_room_history ON messages(room_id,id);
CREATE TABLE receipts(message_id INTEGER NOT NULL REFERENCES messages(id),device_id TEXT NOT NULL REFERENCES devices(id),delivered_at INTEGER,read_at INTEGER,withdrawn_at INTEGER,PRIMARY KEY(message_id,device_id));
CREATE INDEX receipts_device_pending ON receipts(device_id,withdrawn_at,delivered_at,message_id);
