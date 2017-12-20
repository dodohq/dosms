DROP TRIGGER IF EXISTS trigger_delete_order
ON orders;
DROP FUNCTION IF EXISTS cascade_order_delete;
DROP TRIGGER IF EXISTS trigger_delete_time_slot
ON time_slots;
DROP FUNCTION IF EXISTS cascade_time_slot_delete;
DROP TRIGGER IF EXISTS trigger_delete_provider
ON providers;
DROP FUNCTION IF EXISTS cascade_provider_delete;
DROP INDEX IF EXISTS index_unique_choice;
DROP TABLE IF EXISTS choices;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS time_slots;
DROP TABLE IF EXISTS providers;
