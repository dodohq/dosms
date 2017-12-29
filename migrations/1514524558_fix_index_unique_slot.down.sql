DROP INDEX IF EXISTS index_unique_time_slot;
DELETE FROM time_slots WHERE deleted;
ALTER TABLE time_slots ADD CONSTRAINT unique_slot UNIQUE(provider_id, start_time, end_time, deleted);

