UPDATE orders SET deleted = TRUE;
CREATE UNIQUE INDEX index_unique_customer_contact ON orders (contact_number) WHERE NOT deleted;