
-- +migrate Up
ALTER TABLE schedules ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;


-- +migrate Down
ALTER TABLE schedules DROP COLUMN enabled;
