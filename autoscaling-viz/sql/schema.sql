CREATE TABLE IF NOT EXISTS collector_events (
  id TEXT PRIMARY KEY,
  time TIMESTAMP NOT NULL,
  revision_id TEXT,
  service_id TEXT,
  instance_delta INT
);
-- Indexes on both revision_id and service_id
CREATE INDEX collector_events_rev_idx ON collector_events (revision_id);
CREATE INDEX collector_events_svc_idx ON collector_events (service_id);

CREATE TABLE IF NOT EXISTS loader_config (  
  href TEXT NOT NULL,
  method TEXT NOT NULL, 
  body TEXT NOT NULL
);

-- Used to track and command loaders
CREATE TABLE IF NOT EXISTS loader_instances (
  id SERIAL primary KEY,
  concurrency INT NOT NULL DEFAULT 0,    -- How many threads are sending requests in a loop    
  reset BOOL NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS loader_request_totals (
  loader_instance_id INT PRIMARY KEY, 
  total_requests INT NOT NULL DEFAULT 0,  -- Total requests handled 
  failed_requests INT NOT NULL DEFAULT 0, -- Total requests non-200 
  rate_per_second INT NOT NULL DEFAULT 0, -- Success request rate per second
  duration REAL NOT NULL DEFAULT 0.0      -- Approx. recent request latency 
);

-- Config services:
INSERT INTO loader_config ( href, method, body ) VALUES (
  'https://rater-kgsrobqqua-ew.a.run.app/vote',
  'POST',
  '{
		"id": "Amsterdam",
		"vote": 1
	}'
);
