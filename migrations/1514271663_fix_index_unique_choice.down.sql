DROP INDEX IF EXISTS index_unique_choice;
CREATE UNIQUE INDEX index_unique_choice ON choices (time_slot_id, order_id, deleted);