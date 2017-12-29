ALTER TABLE time_slots DROP CONSTRAINT IF EXISTS unique_slot;
DELETE FROM time_slots WHERE deleted;
CREATE UNIQUE INDEX index_unique_time_slot on time_slots(provider_id, start_time, end_time) WHERE NOT deleted;
