
-- +migrate Up
CREATE TABLE IF NOT EXISTS schedules (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	app_name TEXT NOT NULL,
	schedule TEXT NOT NULL,
	command TEXT NOT NULL,
	region TEXT NOT NULL,
	config JSON NOT NULL,
	UNIQUE(name)
);

-- +migrate Down
DROP TABLE schedules;
