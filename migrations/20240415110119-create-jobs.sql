
-- +migrate Up
CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL,
    status TEXT CHECK(status IN ('pending', 'running', 'completed', 'failed')) NOT NULL DEFAULT 'pending',
    machine_id TEXT,
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP,
    FOREIGN KEY(schedule_id) REFERENCES schedules(id)
);
-- +migrate Down
DROP TABLE jobs;
