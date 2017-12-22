CREATE TABLE IF NOT EXISTS providers (
  id SERIAL,
  title VARCHAR(45) NOT NULL UNIQUE,
  contact_number VARCHAR(20) NOT NULL UNIQUE,
  deleted BOOLEAN DEFAULT FALSE,
  PRIMARY KEY(id)
);
CREATE TABLE IF NOT EXISTS time_slots (
  id SERIAL,
  start_time TIMESTAMP WITH TIME ZONE NOT NULL,
  end_time TIMESTAMP WITH TIME ZONE NOT NULL,
  provider_id INT NOT NULL,
  deleted BOOLEAN DEFAULT FALSE,
  PRIMARY KEY (id),
  CONSTRAINT unique_slot UNIQUE(provider_id, start_time, end_time, deleted),
  FOREIGN KEY(provider_id) REFERENCES providers(id)
);
CREATE TABLE IF NOT EXISTS orders (
  id SERIAL,
  customer_name VARCHAR(255) NOT NULL,
  contact_number VARCHAR(20) NOT NULL,
  provider_id INT NOT NULL,
  deleted BOOLEAN DEFAULT FALSE,
  PRIMARY KEY(id),
  FOREIGN KEY(provider_id) REFERENCES providers(id)
);
CREATE TABLE IF NOT EXISTS choices (
  time_slot_id INT NOT NULL,
  order_id INT NOT NULL,
  deleted BOOLEAN DEFAULT FALSE,
  FOREIGN KEY(time_slot_id) REFERENCES time_slots(id),
  FOREIGN KEY(order_id) REFERENCES orders(id)
);
CREATE UNIQUE INDEX index_unique_choice ON choices (time_slot_id, order_id, deleted);
CREATE OR REPLACE FUNCTION cascade_provider_delete()
RETURNS trigger AS 
$BODY$
BEGIN
  UPDATE time_slots
  SET deleted = NEW.deleted
  WHERE provider_id = NEW.id;
  UPDATE orders 
  SET deleted = NEW.deleted
  WHERE provider_id = NEW.id;
  RETURN NEW;
END;
$BODY$
LANGUAGE plpgsql;
CREATE TRIGGER trigger_delete_provider 
BEFORE UPDATE ON providers 
FOR EACH ROW
WHEN (OLD.deleted IS DISTINCT FROM NEW.deleted)
EXECUTE PROCEDURE cascade_provider_delete();
CREATE OR REPLACE FUNCTION cascade_time_slot_delete()
RETURNS trigger AS 
$BODY$
BEGIN
  UPDATE choices
  SET deleted = NEW.deleted
  WHERE time_slot_id = NEW.id;
  RETURN NEW;
END;
$BODY$
LANGUAGE plpgsql;
CREATE TRIGGER trigger_delete_time_slot 
BEFORE UPDATE ON time_slots 
FOR EACH ROW
WHEN (OLD.deleted IS DISTINCT FROM NEW.deleted)
EXECUTE PROCEDURE cascade_time_slot_delete();
CREATE OR REPLACE FUNCTION cascade_order_delete()
RETURNS trigger AS 
$BODY$
BEGIN
  UPDATE choices
  SET deleted = NEW.deleted
  WHERE order_id = NEW.id;
  RETURN NEW;
END;
$BODY$
LANGUAGE plpgsql;
CREATE TRIGGER trigger_delete_order 
BEFORE UPDATE ON orders FOR EACH ROW
WHEN (OLD.deleted IS DISTINCT FROM NEW.deleted)
EXECUTE PROCEDURE cascade_order_delete();
