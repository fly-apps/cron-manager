
-- +migrate Up
ALTER TABLE schedules ADD COLUMN command_timeout INTEGER NOT NULL DEFAULT 30;

-- +migrate Down
ALTER TABLE schedules DROP COLUMN command_timeout;
