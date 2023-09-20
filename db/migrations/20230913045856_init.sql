-- migrate:up
CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  type TEXT,
  actor TEXT,
  repo TEXT,
  payload TEXT,
  org TEXT,
  created_at TEXT
)

-- migrate:down
DROP TABLE EVENTS
