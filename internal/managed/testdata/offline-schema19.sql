-- Hand-authored schema19 compatibility fixture, not a claimed historical binary
-- qualification archive. No actual piers, Docker resources or credentials.
PRAGMA user_version=19;
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE instances (id TEXT PRIMARY KEY, body BLOB NOT NULL);
CREATE TABLE operations (id TEXT PRIMARY KEY, actor TEXT NOT NULL, request_id TEXT NOT NULL,
 fingerprint TEXT NOT NULL, body BLOB NOT NULL, runtime BLOB, UNIQUE(actor,request_id));
CREATE TABLE maintenance_policies (instance_id TEXT PRIMARY KEY REFERENCES instances(id), body BLOB NOT NULL);
CREATE TABLE deleted_instances (id TEXT PRIMARY KEY, operation_id TEXT NOT NULL, revision INTEGER NOT NULL);
INSERT INTO meta VALUES ('runner_id','runner-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa');
INSERT INTO meta VALUES ('fixture-private-state','retain-this-byte-string');
INSERT INTO instances VALUES ('running', '{"public":{"id":"running","ship":"~zod","config_revision":3,"applied_revision":3,"desired_state":"running","observed_state":"running","pier_path":"@ssd/fakeships/running/zod","config":{"http_port":19080,"loom":31,"native_control":true}},"runtime":{"id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","port":19080,"prepared":true,"initialized":true,"loom":31,"native_control":true,"runtime":{"containers":[{"id":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","name":"retained-runtime","kind":"http-restart","create_dispatched":true,"alive_observed":true}]}},"runtime_revision":8,"runtime_config_revision":3}');
INSERT INTO instances VALUES ('stopped', '{"public":{"id":"stopped","ship":"~zod","config_revision":5,"desired_state":"stopped","observed_state":"stopped","pier_path":"@hdd/fakeships/stopped/zod","config":{"http_port":19081,"loom":30}},"runtime":{"id":"dddddddddddddddddddddddddddddddd","port":19081,"prepared":true,"initialized":true,"loom":30,"runtime":{"containers":[]}},"runtime_revision":9,"runtime_config_revision":5}');
INSERT INTO operations VALUES ('op-old','ssh-key:fixture-owner','old-request','original-fingerprint','{"id":"op-old","actor":"ssh-key:fixture-owner","request_id":"old-request","instance_id":"running","kind":"start","status":"succeeded","config_revision":3,"created_at":"2026-10-10T00:00:00Z","finished_at":"2026-10-10T00:01:00Z"}',X'7B7D');
INSERT INTO maintenance_policies VALUES ('stopped','{"id":"policy-retained","instance_id":"stopped","revision":2,"instance_revision":5,"owner":"ssh-key:fixture-owner","config":{"enabled":false,"action":"pack","weekdays":[1],"hour_utc":2,"minute_utc":30,"window_minutes":30,"stop_running":false,"resume":false},"updated_at":"2026-10-10T00:00:00Z"}');
INSERT INTO deleted_instances VALUES ('retired','op-retire',7);
