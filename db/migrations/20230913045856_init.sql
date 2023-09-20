-- migrate:up
CREATE TABLE IF NOT EXISTS events (
  id VARCHAR(255) PRIMARY KEY,
  type VARCHAR(255),
  actor VARCHAR(255),
  repo VARCHAR(255),
  payload JSON,
  org VARCHAR(255),
  created_at DATETIME
);

-- migrate:down
DROP TABLE EVENTS;
