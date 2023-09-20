CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE events (
  id TEXT PRIMARY KEY,
  type TEXT,
  actor TEXT,
  repo TEXT,
  payload TEXT,
  org TEXT,
  created_at TEXT
);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20230913045856');
